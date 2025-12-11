# Frontend Reports Dashboard - Guia de Implementação

Este guia detalha como implementar o dashboard de reports no frontend usando **Lovable** e **Supabase**.

---

## 📊 Visão Geral

O sistema de reports fornece métricas detalhadas sobre:
- **Uso de tokens** por operação de IA
- **Custos estimados** em USD
- **Estatísticas de geração de leads**
- **Métricas diárias** para gráficos

---

## 🗄️ Estrutura de Dados no Supabase

### Tabela `usage_metrics`

```sql
CREATE TABLE usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id),
    job_id UUID REFERENCES jobs(id),
    lead_id UUID REFERENCES leads(id),
    operation_type operation_type NOT NULL, -- 'data_extraction', 'pre_call_report', 'cold_email', 'website_scraping'
    model TEXT NOT NULL,                    -- 'gemini-2.5-flash', 'gemini-2.5-pro'
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10, 6) NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Enum `operation_type`

```typescript
type OperationType = 
  | 'data_extraction'    // Extração de dados de websites
  | 'pre_call_report'    // Geração de relatórios pré-chamada
  | 'cold_email'         // Geração de cold emails
  | 'website_scraping';  // Scraping de websites (futuro)
```

---

## 🔌 API Endpoints Disponíveis

### Base URL
```
https://your-api-domain.com/api/v1
```

### 1. GET `/api/v1/reports` - Relatório Completo

**Query Parameters:**
| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `user_id` | string | ✅ | UUID do usuário |
| `start_date` | string | ❌ | Data início (YYYY-MM-DD) |
| `end_date` | string | ❌ | Data fim (YYYY-MM-DD) |

**Response:**
```json
{
  "period": {
    "start_date": "2024-01-01T00:00:00Z",
    "end_date": "2024-01-31T23:59:59Z"
  },
  "summary": {
    "total_operations": 1250,
    "total_input_tokens": 2500000,
    "total_output_tokens": 500000,
    "total_tokens": 3000000,
    "total_cost_usd": 4.50,
    "success_rate": 98.5,
    "avg_duration_ms": 2500
  },
  "by_operation": [
    {
      "operation_type": "data_extraction",
      "count": 500,
      "total_tokens": 1200000,
      "total_cost_usd": 1.80,
      "success_rate": 99.0,
      "avg_duration_ms": 2000
    },
    {
      "operation_type": "pre_call_report",
      "count": 400,
      "total_tokens": 1000000,
      "total_cost_usd": 1.50,
      "success_rate": 98.0,
      "avg_duration_ms": 3000
    },
    {
      "operation_type": "cold_email",
      "count": 350,
      "total_tokens": 800000,
      "total_cost_usd": 1.20,
      "success_rate": 98.5,
      "avg_duration_ms": 2500
    }
  ],
  "by_model": [
    {
      "model": "gemini-2.5-flash",
      "count": 1200,
      "total_tokens": 2800000,
      "total_cost_usd": 0.70
    },
    {
      "model": "gemini-2.5-pro",
      "count": 50,
      "total_tokens": 200000,
      "total_cost_usd": 3.80
    }
  ],
  "daily_usage": [
    {
      "date": "2024-01-15",
      "operations": 45,
      "tokens": 100000,
      "cost_usd": 0.15
    }
  ],
  "lead_generation": {
    "total_leads_generated": 250,
    "total_reports_generated": 200,
    "total_emails_generated": 180,
    "leads_with_contact": 220,
    "conversion_rate": 88.0
  }
}
```

### 2. GET `/api/v1/reports/summary` - Resumo Rápido

**Response:**
```json
{
  "total_operations": 1250,
  "total_tokens": 3000000,
  "total_cost_usd": 4.50,
  "success_rate": 98.5
}
```

### 3. GET `/api/v1/reports/daily` - Uso Diário (para gráficos)

**Response:**
```json
{
  "daily_usage": [
    {
      "date": "2024-01-01",
      "operations": 40,
      "tokens": 95000,
      "cost_usd": 0.14
    },
    {
      "date": "2024-01-02",
      "operations": 52,
      "tokens": 120000,
      "cost_usd": 0.18
    }
  ]
}
```

### 4. GET `/api/v1/reports/operations` - Por Tipo de Operação

**Response:**
```json
{
  "by_operation": [
    {
      "operation_type": "data_extraction",
      "count": 500,
      "total_tokens": 1200000,
      "total_cost_usd": 1.80,
      "success_rate": 99.0,
      "avg_duration_ms": 2000
    }
  ]
}
```

---

## 🎨 Componentes do Dashboard

### 1. Cards de Resumo (Summary Cards)

Crie 4 cards principais no topo do dashboard:

```tsx
// components/reports/SummaryCards.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Activity, Coins, Zap, CheckCircle } from "lucide-react";

interface SummaryCardsProps {
  summary: {
    total_operations: number;
    total_tokens: number;
    total_cost_usd: number;
    success_rate: number;
  };
}

export function SummaryCards({ summary }: SummaryCardsProps) {
  const cards = [
    {
      title: "Total de Operações",
      value: summary.total_operations.toLocaleString(),
      icon: Activity,
      color: "text-blue-600",
      bgColor: "bg-blue-100",
    },
    {
      title: "Tokens Processados",
      value: formatTokens(summary.total_tokens),
      icon: Zap,
      color: "text-yellow-600",
      bgColor: "bg-yellow-100",
    },
    {
      title: "Custo Estimado",
      value: `$${summary.total_cost_usd.toFixed(2)}`,
      icon: Coins,
      color: "text-green-600",
      bgColor: "bg-green-100",
    },
    {
      title: "Taxa de Sucesso",
      value: `${summary.success_rate.toFixed(1)}%`,
      icon: CheckCircle,
      color: "text-emerald-600",
      bgColor: "bg-emerald-100",
    },
  ];

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
            <div className={`p-2 rounded-full ${card.bgColor}`}>
              <card.icon className={`h-4 w-4 ${card.color}`} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{card.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function formatTokens(tokens: number): string {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(1)}M`;
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`;
  }
  return tokens.toString();
}
```

### 2. Gráfico de Uso Diário (Area Chart)

```tsx
// components/reports/DailyUsageChart.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface DailyUsage {
  date: string;
  operations: number;
  tokens: number;
  cost_usd: number;
}

interface DailyUsageChartProps {
  data: DailyUsage[];
}

export function DailyUsageChart({ data }: DailyUsageChartProps) {
  const formattedData = data.map((item) => ({
    ...item,
    date: new Date(item.date).toLocaleDateString("pt-BR", {
      day: "2-digit",
      month: "short",
    }),
    tokens_k: item.tokens / 1000,
  }));

  return (
    <Card className="col-span-4">
      <CardHeader>
        <CardTitle>Uso Diário de Tokens</CardTitle>
      </CardHeader>
      <CardContent className="h-[300px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={formattedData}>
            <defs>
              <linearGradient id="colorTokens" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#8884d8" stopOpacity={0.8} />
                <stop offset="95%" stopColor="#8884d8" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" />
            <YAxis tickFormatter={(value) => `${value}K`} />
            <Tooltip
              formatter={(value: number) => [`${value.toFixed(1)}K tokens`]}
              labelFormatter={(label) => `Data: ${label}`}
            />
            <Area
              type="monotone"
              dataKey="tokens_k"
              stroke="#8884d8"
              fillOpacity={1}
              fill="url(#colorTokens)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
```

### 3. Gráfico de Pizza por Operação

```tsx
// components/reports/OperationsPieChart.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from "recharts";

interface OperationStats {
  operation_type: string;
  count: number;
  total_tokens: number;
  total_cost_usd: number;
}

interface OperationsPieChartProps {
  data: OperationStats[];
}

const COLORS = ["#0088FE", "#00C49F", "#FFBB28", "#FF8042"];

const OPERATION_LABELS: Record<string, string> = {
  data_extraction: "Extração de Dados",
  pre_call_report: "Relatórios Pré-Chamada",
  cold_email: "Cold Emails",
  website_scraping: "Web Scraping",
};

export function OperationsPieChart({ data }: OperationsPieChartProps) {
  const chartData = data.map((item) => ({
    name: OPERATION_LABELS[item.operation_type] || item.operation_type,
    value: item.count,
    tokens: item.total_tokens,
    cost: item.total_cost_usd,
  }));

  return (
    <Card className="col-span-2">
      <CardHeader>
        <CardTitle>Operações por Tipo</CardTitle>
      </CardHeader>
      <CardContent className="h-[300px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              labelLine={false}
              outerRadius={80}
              fill="#8884d8"
              dataKey="value"
              label={({ name, percent }) =>
                `${name} (${(percent * 100).toFixed(0)}%)`
              }
            >
              {chartData.map((entry, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={COLORS[index % COLORS.length]}
                />
              ))}
            </Pie>
            <Tooltip
              formatter={(value: number, name: string, props: any) => [
                `${value} operações`,
                name,
              ]}
            />
            <Legend />
          </PieChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}
```

### 4. Tabela de Uso por Modelo

```tsx
// components/reports/ModelUsageTable.tsx
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

interface ModelUsage {
  model: string;
  count: number;
  total_tokens: number;
  total_cost_usd: number;
}

interface ModelUsageTableProps {
  data: ModelUsage[];
}

export function ModelUsageTable({ data }: ModelUsageTableProps) {
  return (
    <Card className="col-span-2">
      <CardHeader>
        <CardTitle>Uso por Modelo</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Modelo</TableHead>
              <TableHead className="text-right">Operações</TableHead>
              <TableHead className="text-right">Tokens</TableHead>
              <TableHead className="text-right">Custo</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((item) => (
              <TableRow key={item.model}>
                <TableCell>
                  <Badge variant={item.model.includes("flash") ? "default" : "secondary"}>
                    {item.model}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">
                  {item.count.toLocaleString()}
                </TableCell>
                <TableCell className="text-right">
                  {(item.total_tokens / 1000).toFixed(1)}K
                </TableCell>
                <TableCell className="text-right font-medium">
                  ${item.total_cost_usd.toFixed(4)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
```

### 5. Estatísticas de Lead Generation

```tsx
// components/reports/LeadGenerationStats.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Users, FileText, Mail, UserCheck } from "lucide-react";

interface LeadGenerationStatsProps {
  stats: {
    total_leads_generated: number;
    total_reports_generated: number;
    total_emails_generated: number;
    leads_with_contact: number;
    conversion_rate: number;
  };
}

export function LeadGenerationStats({ stats }: LeadGenerationStatsProps) {
  const items = [
    {
      label: "Leads Gerados",
      value: stats.total_leads_generated,
      icon: Users,
    },
    {
      label: "Relatórios Gerados",
      value: stats.total_reports_generated,
      icon: FileText,
    },
    {
      label: "Emails Gerados",
      value: stats.total_emails_generated,
      icon: Mail,
    },
    {
      label: "Leads com Contato",
      value: stats.leads_with_contact,
      icon: UserCheck,
    },
  ];

  return (
    <Card className="col-span-4">
      <CardHeader>
        <CardTitle>Geração de Leads</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {items.map((item) => (
            <div key={item.label} className="flex items-center gap-4">
              <div className="p-2 bg-primary/10 rounded-full">
                <item.icon className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{item.label}</p>
                <p className="text-2xl font-bold">{item.value.toLocaleString()}</p>
              </div>
            </div>
          ))}
        </div>
        
        <div className="mt-6">
          <div className="flex justify-between mb-2">
            <span className="text-sm font-medium">Taxa de Conversão</span>
            <span className="text-sm font-medium">{stats.conversion_rate.toFixed(1)}%</span>
          </div>
          <Progress value={stats.conversion_rate} className="h-2" />
        </div>
      </CardContent>
    </Card>
  );
}
```

---

## 🔧 Hook para Buscar Dados

```tsx
// hooks/useReports.ts
import { useState, useEffect } from "react";
import { useAuth } from "@/hooks/useAuth"; // Seu hook de autenticação

interface ReportsData {
  period: {
    start_date: string;
    end_date: string;
  };
  summary: {
    total_operations: number;
    total_input_tokens: number;
    total_output_tokens: number;
    total_tokens: number;
    total_cost_usd: number;
    success_rate: number;
    avg_duration_ms: number;
  };
  by_operation: Array<{
    operation_type: string;
    count: number;
    total_tokens: number;
    total_cost_usd: number;
    success_rate: number;
    avg_duration_ms: number;
  }>;
  by_model: Array<{
    model: string;
    count: number;
    total_tokens: number;
    total_cost_usd: number;
  }>;
  daily_usage: Array<{
    date: string;
    operations: number;
    tokens: number;
    cost_usd: number;
  }>;
  lead_generation: {
    total_leads_generated: number;
    total_reports_generated: number;
    total_emails_generated: number;
    leads_with_contact: number;
    conversion_rate: number;
  };
}

interface UseReportsOptions {
  startDate?: string;
  endDate?: string;
}

export function useReports(options: UseReportsOptions = {}) {
  const { user } = useAuth();
  const [data, setData] = useState<ReportsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!user?.id) {
      setLoading(false);
      return;
    }

    async function fetchReports() {
      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams({
          user_id: user.id,
        });

        if (options.startDate) {
          params.append("start_date", options.startDate);
        }
        if (options.endDate) {
          params.append("end_date", options.endDate);
        }

        const response = await fetch(
          `${import.meta.env.VITE_API_URL}/api/v1/reports?${params}`
        );

        if (!response.ok) {
          throw new Error("Failed to fetch reports");
        }

        const result = await response.json();
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    }

    fetchReports();
  }, [user?.id, options.startDate, options.endDate]);

  return { data, loading, error, refetch: () => {} };
}
```

---

## 📱 Página Principal do Dashboard

```tsx
// pages/Reports.tsx
import { useState } from "react";
import { useReports } from "@/hooks/useReports";
import { SummaryCards } from "@/components/reports/SummaryCards";
import { DailyUsageChart } from "@/components/reports/DailyUsageChart";
import { OperationsPieChart } from "@/components/reports/OperationsPieChart";
import { ModelUsageTable } from "@/components/reports/ModelUsageTable";
import { LeadGenerationStats } from "@/components/reports/LeadGenerationStats";
import { DateRangePicker } from "@/components/ui/date-range-picker";
import { Skeleton } from "@/components/ui/skeleton";
import { AlertCircle } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

export default function ReportsPage() {
  const [dateRange, setDateRange] = useState<{
    from: Date | undefined;
    to: Date | undefined;
  }>({
    from: undefined,
    to: undefined,
  });

  const { data, loading, error } = useReports({
    startDate: dateRange.from?.toISOString().split("T")[0],
    endDate: dateRange.to?.toISOString().split("T")[0],
  });

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Erro</AlertTitle>
        <AlertDescription>
          Não foi possível carregar os relatórios. Tente novamente mais tarde.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">Reports</h2>
        <div className="flex items-center space-x-2">
          <DateRangePicker
            value={dateRange}
            onChange={setDateRange}
          />
        </div>
      </div>

      {loading ? (
        <ReportsPageSkeleton />
      ) : data ? (
        <div className="space-y-4">
          {/* Summary Cards */}
          <SummaryCards summary={data.summary} />

          {/* Charts Row */}
          <div className="grid gap-4 md:grid-cols-4">
            <DailyUsageChart data={data.daily_usage} />
          </div>

          {/* Operations & Models Row */}
          <div className="grid gap-4 md:grid-cols-4">
            <OperationsPieChart data={data.by_operation} />
            <ModelUsageTable data={data.by_model} />
          </div>

          {/* Lead Generation Stats */}
          <div className="grid gap-4 md:grid-cols-4">
            <LeadGenerationStats stats={data.lead_generation} />
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ReportsPageSkeleton() {
  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-[120px]" />
        ))}
      </div>
      <Skeleton className="h-[350px]" />
      <div className="grid gap-4 md:grid-cols-4">
        <Skeleton className="h-[350px] col-span-2" />
        <Skeleton className="h-[350px] col-span-2" />
      </div>
    </div>
  );
}
```

---

## 🔐 Autenticação e Permissões

### Middleware de Autenticação

O backend valida o `user_id` via JWT do Supabase. No frontend:

```tsx
// Exemplo de chamada autenticada
const { data: session } = await supabase.auth.getSession();

const response = await fetch(`${API_URL}/api/v1/reports?user_id=${session.user.id}`, {
  headers: {
    'Authorization': `Bearer ${session.access_token}`,
  },
});
```

### RLS no Supabase

A tabela `usage_metrics` tem RLS habilitado:
- Usuários só veem suas próprias métricas
- Service role pode inserir/ler tudo

---

## 📦 Dependências Necessárias

```bash
npm install recharts lucide-react date-fns
```

Componentes shadcn/ui necessários:
```bash
npx shadcn-ui@latest add card table badge progress alert skeleton
```

---

## 🎯 Checklist de Implementação

- [ ] Criar página `/reports` no roteamento
- [ ] Implementar hook `useReports`
- [ ] Criar componente `SummaryCards`
- [ ] Criar componente `DailyUsageChart`
- [ ] Criar componente `OperationsPieChart`
- [ ] Criar componente `ModelUsageTable`
- [ ] Criar componente `LeadGenerationStats`
- [ ] Adicionar filtro de data (DateRangePicker)
- [ ] Adicionar loading states (Skeleton)
- [ ] Adicionar tratamento de erros
- [ ] Testar responsividade
- [ ] Adicionar link no menu de navegação

---

## 🎨 Design System

### Cores Sugeridas

```css
/* Operações */
--color-extraction: #0088FE;
--color-reports: #00C49F;
--color-emails: #FFBB28;
--color-scraping: #FF8042;

/* Status */
--color-success: #10B981;
--color-warning: #F59E0B;
--color-error: #EF4444;
```

### Layout Responsivo

```
Desktop (lg+):
┌──────────┬──────────┬──────────┬──────────┐
│  Card 1  │  Card 2  │  Card 3  │  Card 4  │
├──────────┴──────────┴──────────┴──────────┤
│              Daily Usage Chart             │
├──────────────────────┬────────────────────┤
│    Operations Pie    │   Model Table      │
├──────────────────────┴────────────────────┤
│          Lead Generation Stats            │
└───────────────────────────────────────────┘

Mobile (sm):
┌──────────┐
│  Card 1  │
├──────────┤
│  Card 2  │
├──────────┤
│    ...   │
└──────────┘
```

---

## 🚀 Prompt para Lovable

Use este prompt no Lovable para gerar a página:

```
Crie uma página de Reports/Dashboard com as seguintes características:

1. **Header** com título "Reports" e um DateRangePicker para filtrar período

2. **4 Cards de Resumo** em grid responsivo mostrando:
   - Total de Operações (ícone Activity)
   - Tokens Processados (ícone Zap) 
   - Custo Estimado em USD (ícone Coins)
   - Taxa de Sucesso % (ícone CheckCircle)

3. **Gráfico de Área** mostrando uso diário de tokens (últimos 30 dias)

4. **Gráfico de Pizza** mostrando distribuição por tipo de operação:
   - data_extraction (azul)
   - pre_call_report (verde)
   - cold_email (amarelo)

5. **Tabela** mostrando uso por modelo de IA com colunas:
   - Modelo (badge colorido)
   - Operações
   - Tokens
   - Custo

6. **Card de Lead Generation** mostrando:
   - Leads gerados
   - Relatórios gerados
   - Emails gerados
   - Progress bar de taxa de conversão

Use shadcn/ui, Recharts para gráficos, Lucide para ícones.
Buscar dados de API: GET /api/v1/reports?user_id={userId}
Layout responsivo com Tailwind CSS.
```

---

## 📞 Suporte

Para dúvidas sobre a API, consulte:
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Documentação: `CLAUDE.md`
