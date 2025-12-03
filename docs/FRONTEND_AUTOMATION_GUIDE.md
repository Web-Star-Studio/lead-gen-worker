# Frontend Automation Implementation Guide

Este documento descreve como implementar o sistema de automação de leads no frontend, incluindo configurações de usuário, disparo de tarefas em lote e acompanhamento em tempo real.

## Tabelas Supabase

### automation_configs
Configurações de automação do usuário.

```typescript
interface AutomationConfig {
  id: string;
  user_id: string;
  auto_enrich_new_leads: boolean;    // Auto-enriquecer leads novos
  auto_generate_precall: boolean;    // Auto-gerar pre-call report
  auto_generate_email: boolean;      // Auto-gerar cold email
  default_business_profile_id: string | null;
  daily_automation_limit: number;    // Limite diário (default: 100)
  created_at: string;
  updated_at: string;
}
```

### automation_tasks
Fila de tarefas de automação.

```typescript
interface AutomationTask {
  id: string;
  user_id: string;
  task_type: 'lead_enrichment' | 'precall_generation' | 'email_generation' | 'full_enrichment';
  lead_id: string | null;           // Lead único
  lead_ids: string[];               // Batch de leads
  business_profile_id: string | null;
  priority: 1 | 2 | 3;              // 1=Alta, 2=Média, 3=Baixa
  status: 'pending' | 'processing' | 'completed' | 'failed';
  items_total: number;
  items_processed: number;
  items_succeeded: number;
  items_failed: number;
  error_message: string | null;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
}
```

## Funcionalidades a Implementar

### 1. Página de Configurações de Automação

**Localização sugerida**: `/settings/automation` ou como tab em `/settings`

**Campos do formulário**:
- Toggle: "Enriquecer novos leads automaticamente"
- Toggle: "Gerar pre-call report automaticamente"
- Toggle: "Gerar cold email automaticamente"
- Select: "Perfil de negócio padrão" (lista de business_profiles)
- Input numérico: "Limite diário de automações"

**Comportamento**:
- Ao carregar: buscar config existente com `supabase.from('automation_configs').select().eq('user_id', userId).single()`
- Se não existir: criar registro com valores padrão
- Ao salvar: fazer upsert no registro

```typescript
// Hook sugerido: useAutomationConfig
const useAutomationConfig = () => {
  const { data: config, isLoading } = useQuery({
    queryKey: ['automation-config'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('automation_configs')
        .select('*')
        .eq('user_id', user.id)
        .single();
      
      if (error && error.code === 'PGRST116') {
        // Não existe, criar padrão
        const { data: newConfig } = await supabase
          .from('automation_configs')
          .insert({ user_id: user.id })
          .select()
          .single();
        return newConfig;
      }
      return data;
    }
  });
  
  const updateConfig = useMutation({
    mutationFn: async (updates: Partial<AutomationConfig>) => {
      const { data, error } = await supabase
        .from('automation_configs')
        .update(updates)
        .eq('user_id', user.id)
        .select()
        .single();
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-config'] });
    }
  });

  return { config, isLoading, updateConfig };
};
```

### 2. Ações em Lote na Lista de Leads

**Localização**: Toolbar da tabela de leads (quando leads selecionados)

**Botões de ação**:
- "🔍 Enriquecer Selecionados" → `task_type: 'lead_enrichment'`
- "📄 Gerar Pre-Call" → `task_type: 'precall_generation'`
- "✉️ Gerar Emails" → `task_type: 'email_generation'`
- "🚀 Enriquecimento Completo" → `task_type: 'full_enrichment'`

**Fluxo**:
1. Usuário seleciona leads na tabela
2. Clica em ação desejada
3. Modal de confirmação com:
   - Quantidade de leads selecionados
   - Tipo de ação
   - Seletor de business_profile (para pre-call/email)
4. Ao confirmar: inserir registro em `automation_tasks`
5. Worker processa automaticamente (via webhook)

```typescript
// Hook sugerido: useCreateAutomationTask
const useCreateAutomationTask = () => {
  return useMutation({
    mutationFn: async (task: {
      task_type: AutomationTask['task_type'];
      lead_ids: string[];
      business_profile_id?: string;
    }) => {
      const { data, error } = await supabase
        .from('automation_tasks')
        .insert({
          user_id: user.id,
          task_type: task.task_type,
          lead_ids: task.lead_ids,
          business_profile_id: task.business_profile_id,
          priority: 3, // Low priority for manual batch
          status: 'pending',
          items_total: task.lead_ids.length,
        })
        .select()
        .single();
      
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automation-tasks'] });
      toast.success('Tarefa de automação criada!');
    }
  });
};
```

### 3. Acompanhamento de Tarefas em Tempo Real

**Localização sugerida**: 
- Sidebar com badge de tarefas ativas
- Página `/automations` ou `/tasks` com histórico
- Toast/notificação ao completar

**Componentes**:

#### TaskProgressCard
Mostra progresso de uma tarefa individual.

```tsx
interface TaskProgressCardProps {
  task: AutomationTask;
}

const TaskProgressCard = ({ task }: TaskProgressCardProps) => {
  const progress = task.items_total > 0 
    ? (task.items_processed / task.items_total) * 100 
    : 0;

  return (
    <div className="p-4 border rounded-lg">
      <div className="flex justify-between items-center mb-2">
        <span className="font-medium">{getTaskTypeLabel(task.task_type)}</span>
        <Badge variant={getStatusVariant(task.status)}>{task.status}</Badge>
      </div>
      
      <Progress value={progress} className="mb-2" />
      
      <div className="text-sm text-muted-foreground">
        {task.items_succeeded} ✓ / {task.items_failed} ✗ de {task.items_total}
      </div>
    </div>
  );
};
```

#### Real-time Subscription Hook

```typescript
const useAutomationTasksRealtime = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const channel = supabase
      .channel('automation-tasks-changes')
      .on(
        'postgres_changes',
        {
          event: '*',
          schema: 'public',
          table: 'automation_tasks',
          filter: `user_id=eq.${user.id}`,
        },
        (payload) => {
          console.log('Task update:', payload);
          
          // Atualizar cache do React Query
          queryClient.setQueryData(
            ['automation-tasks'],
            (old: AutomationTask[] | undefined) => {
              if (!old) return [payload.new as AutomationTask];
              
              if (payload.eventType === 'INSERT') {
                return [payload.new as AutomationTask, ...old];
              }
              
              if (payload.eventType === 'UPDATE') {
                return old.map((task) =>
                  task.id === (payload.new as AutomationTask).id
                    ? (payload.new as AutomationTask)
                    : task
                );
              }
              
              return old;
            }
          );
          
          // Toast quando completar
          if (payload.new.status === 'completed') {
            toast.success(
              `✓ ${payload.new.items_succeeded} leads processados com sucesso!`
            );
          }
          
          if (payload.new.status === 'failed') {
            toast.error(
              `Tarefa falhou: ${payload.new.error_message}`
            );
          }
        }
      )
      .subscribe();

    return () => {
      supabase.removeChannel(channel);
    };
  }, [user.id, queryClient]);
};
```

### 4. Atualização em Tempo Real dos Leads

Quando leads são enriquecidos, eles recebem novos dados. O frontend deve escutar essas mudanças:

```typescript
const useLeadsRealtime = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const channel = supabase
      .channel('leads-changes')
      .on(
        'postgres_changes',
        {
          event: 'UPDATE',
          schema: 'public',
          table: 'leads',
          filter: `user_id=eq.${user.id}`,
        },
        (payload) => {
          // Atualizar lead específico no cache
          queryClient.setQueryData(
            ['leads'],
            (old: Lead[] | undefined) => {
              if (!old) return old;
              return old.map((lead) =>
                lead.id === payload.new.id ? payload.new : lead
              );
            }
          );
        }
      )
      .subscribe();

    return () => {
      supabase.removeChannel(channel);
    };
  }, [user.id, queryClient]);
};
```

## Estrutura de Arquivos Sugerida

```
src/
├── features/
│   └── automation/
│       ├── hooks/
│       │   ├── useAutomationConfig.ts
│       │   ├── useCreateAutomationTask.ts
│       │   ├── useAutomationTasks.ts
│       │   └── useAutomationTasksRealtime.ts
│       ├── components/
│       │   ├── AutomationSettingsForm.tsx
│       │   ├── BatchActionsToolbar.tsx
│       │   ├── TaskProgressCard.tsx
│       │   ├── TasksList.tsx
│       │   └── BatchActionModal.tsx
│       ├── types/
│       │   └── automation.ts
│       └── pages/
│           └── AutomationSettingsPage.tsx
```

## Checklist de Implementação

### Configurações
- [ ] Criar página de configurações de automação
- [ ] Implementar formulário com toggles
- [ ] Adicionar seletor de business profile padrão
- [ ] Implementar upsert de configuração

### Ações em Lote
- [ ] Adicionar seleção múltipla na tabela de leads
- [ ] Criar toolbar com ações de automação
- [ ] Implementar modal de confirmação
- [ ] Criar mutation para inserir automation_task

### Acompanhamento
- [ ] Criar componente de progresso de tarefa
- [ ] Implementar subscription realtime para tasks
- [ ] Adicionar notificações toast
- [ ] Criar página de histórico de tarefas

### Realtime
- [ ] Implementar subscription para updates de leads
- [ ] Atualizar cache do React Query em tempo real
- [ ] Mostrar indicadores visuais de leads em processamento

## Notas Importantes

1. **RLS está ativo**: As tabelas têm Row Level Security. O frontend só consegue ver dados do próprio usuário.

2. **Prioridades**: 
   - 1 = Alta (jobs de busca, real-time)
   - 2 = Média (auto-enriquecimento)
   - 3 = Baixa (batch manual)

3. **Limite diário**: Considere mostrar quantas automações o usuário já usou hoje vs. o limite.

4. **Estados visuais**:
   - `pending` → Ícone de relógio, cor cinza
   - `processing` → Spinner, cor azul
   - `completed` → Check verde
   - `failed` → X vermelho

5. **Webhook automático**: Ao inserir em `automation_tasks`, o worker é notificado automaticamente via Supabase webhook. Não precisa chamar API externa do frontend.
