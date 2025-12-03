# Prompt para Implementação do Sistema de Automação no Frontend

## Contexto

O backend do lead-gen-worker foi atualizado com um sistema de automação que permite:
1. **Configurações de automação por usuário**: toggles para auto-enriquecer leads, gerar pre-call reports e cold emails automaticamente
2. **Tarefas de automação em lote**: processar múltiplos leads de uma vez
3. **Acompanhamento em tempo real**: ver o progresso das tarefas via Supabase Realtime

Duas novas tabelas foram criadas no Supabase:
- `automation_configs`: configurações de automação do usuário
- `automation_tasks`: fila de tarefas de automação

---

## PROMPT PARA O AGENTE

```
Implemente o sistema de automação de leads no frontend seguindo estas especificações:

## 1. Novas Tabelas Supabase (já criadas no banco)

### automation_configs
```typescript
interface AutomationConfig {
  id: string;
  user_id: string;
  auto_enrich_new_leads: boolean;    // Auto-enriquecer leads novos (scrape + extract)
  auto_generate_precall: boolean;    // Auto-gerar pre-call report
  auto_generate_email: boolean;      // Auto-gerar cold email
  default_business_profile_id: string | null;  // Perfil padrão para automações
  daily_automation_limit: number;    // Limite diário (default: 100)
  created_at: string;
  updated_at: string;
}
```

### automation_tasks
```typescript
interface AutomationTask {
  id: string;
  user_id: string;
  task_type: 'lead_enrichment' | 'precall_generation' | 'email_generation' | 'full_enrichment';
  lead_id: string | null;           // Lead único (opcional)
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

## 2. Funcionalidades a Implementar

### 2.1 Página de Configurações de Automação
Localização: `/settings/automation` ou como tab em `/settings`

Campos:
- Toggle: "Enriquecer novos leads automaticamente" (auto_enrich_new_leads)
- Toggle: "Gerar pre-call report automaticamente" (auto_generate_precall)  
- Toggle: "Gerar cold email automaticamente" (auto_generate_email)
- Select: "Perfil de negócio padrão" → lista de business_profiles do usuário
- Input numérico: "Limite diário de automações" (daily_automation_limit)

Comportamento:
- Ao carregar: buscar config existente ou criar com valores padrão
- Ao salvar: fazer upsert no registro
- Usar React Query para cache e mutations

### 2.2 Ações em Lote na Tabela de Leads
Adicionar toolbar que aparece quando leads estão selecionados com os botões:
- "🔍 Enriquecer" → task_type: 'lead_enrichment'
- "📄 Gerar Pre-Call" → task_type: 'precall_generation'
- "✉️ Gerar Emails" → task_type: 'email_generation'
- "🚀 Enriquecimento Completo" → task_type: 'full_enrichment'

Modal de confirmação deve:
- Mostrar quantidade de leads selecionados
- Permitir selecionar business_profile (para pre-call e email)
- Ao confirmar: INSERT na tabela automation_tasks

### 2.3 Acompanhamento em Tempo Real
Implementar subscription Supabase Realtime para:
1. Tabela `automation_tasks` → atualizar progresso
2. Tabela `leads` → atualizar dados quando enriquecidos

Componentes:
- TaskProgressCard: mostra progresso individual com barra de progresso
- Sidebar badge: contador de tarefas ativas
- Toasts: notificar quando tarefa completa/falha

## 3. Hooks Sugeridos

```typescript
// useAutomationConfig - gerenciar configurações
const useAutomationConfig = () => {
  // Query para buscar config (criar se não existir)
  // Mutation para atualizar config
};

// useCreateAutomationTask - criar tarefa de automação
const useCreateAutomationTask = () => {
  // Mutation para inserir em automation_tasks
  // Invalidar queries após sucesso
};

// useAutomationTasks - listar tarefas do usuário
const useAutomationTasks = () => {
  // Query para listar tarefas recentes
  // Ordenar por created_at DESC
};

// useAutomationTasksRealtime - subscription para updates
const useAutomationTasksRealtime = () => {
  // Subscription Supabase para postgres_changes
  // Atualizar cache React Query em tempo real
  // Mostrar toasts para status completed/failed
};

// useLeadsRealtime - subscription para leads atualizados
const useLeadsRealtime = () => {
  // Subscription para UPDATE em leads
  // Atualizar cache React Query
};
```

## 4. Exemplo de Subscription Realtime

```typescript
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
        // Atualizar React Query cache
        queryClient.setQueryData(['automation-tasks'], (old) => {
          if (payload.eventType === 'INSERT') {
            return [payload.new, ...old];
          }
          if (payload.eventType === 'UPDATE') {
            return old.map((task) =>
              task.id === payload.new.id ? payload.new : task
            );
          }
          return old;
        });
        
        // Toast de notificação
        if (payload.new.status === 'completed') {
          toast.success(`✓ ${payload.new.items_succeeded} leads processados!`);
        }
        if (payload.new.status === 'failed') {
          toast.error(`Erro: ${payload.new.error_message}`);
        }
      }
    )
    .subscribe();

  return () => supabase.removeChannel(channel);
}, [user.id]);
```

## 5. UI/UX Guidelines

### Estados visuais para status de tarefa:
- `pending` → Ícone Clock, Badge cinza
- `processing` → Spinner animado, Badge azul
- `completed` → Ícone CheckCircle, Badge verde
- `failed` → Ícone XCircle, Badge vermelho

### Progress Bar:
```tsx
<Progress 
  value={(task.items_processed / task.items_total) * 100} 
  className={cn(
    task.status === 'failed' && 'bg-red-500',
    task.status === 'completed' && 'bg-green-500'
  )}
/>
```

### Labels para task_type:
```typescript
const taskTypeLabels = {
  lead_enrichment: '🔍 Enriquecimento',
  precall_generation: '📄 Pre-Call Report',
  email_generation: '✉️ Cold Email',
  full_enrichment: '🚀 Enriquecimento Completo',
};
```

## 6. Arquivos a Criar

```
src/features/automation/
├── types/automation.ts           # Interfaces TypeScript
├── hooks/
│   ├── useAutomationConfig.ts
│   ├── useCreateAutomationTask.ts
│   ├── useAutomationTasks.ts
│   └── useAutomationTasksRealtime.ts
├── components/
│   ├── AutomationSettingsForm.tsx
│   ├── BatchActionsToolbar.tsx
│   ├── BatchActionModal.tsx
│   ├── TaskProgressCard.tsx
│   └── TasksList.tsx
└── pages/
    └── AutomationSettingsPage.tsx
```

## 7. Notas Importantes

1. RLS está ativo - usuário só vê seus próprios dados
2. Ao inserir em automation_tasks, o worker é notificado automaticamente via webhook Supabase (não precisa chamar API)
3. Prioridade 3 para tarefas manuais em lote
4. Considerar mostrar uso diário vs limite (daily_automation_limit)

Use shadcn/ui para componentes, React Query para state, e Supabase JS para database/realtime.
```

---

## Referência Rápida

### Inserir Tarefa de Automação
```typescript
await supabase.from('automation_tasks').insert({
  user_id: user.id,
  task_type: 'full_enrichment',
  lead_ids: selectedLeadIds,
  business_profile_id: selectedProfileId,
  priority: 3,
  status: 'pending',
  items_total: selectedLeadIds.length,
});
```

### Buscar Configuração
```typescript
const { data } = await supabase
  .from('automation_configs')
  .select('*')
  .eq('user_id', user.id)
  .single();
```

### Atualizar Configuração
```typescript
await supabase
  .from('automation_configs')
  .upsert({
    user_id: user.id,
    auto_enrich_new_leads: true,
    auto_generate_precall: true,
    auto_generate_email: false,
    default_business_profile_id: profileId,
    daily_automation_limit: 100,
  });
```
