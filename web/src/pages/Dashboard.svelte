<script>
  import AppLayout from '../components/AppLayout.svelte'
  import StatCard from '../components/StatCard.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let stats = null
  export let balances = null
  export let sales = []
  export let events = []
  export let dateRange = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

  /** API amounts are in centavos (minor units). */
  function formatBRL(amount) {
    if (amount == null || amount === '' || Number.isNaN(Number(amount))) return '—'
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(Number(amount) / 100)
  }

  function formatInt(n) {
    if (n == null || Number.isNaN(Number(n))) return '—'
    return new Intl.NumberFormat('pt-BR').format(Number(n))
  }

  function formatDate(iso) {
    if (!iso) return '—'
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return iso
    return d.toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  function statusLabel(status) {
    const map = {
      paid: 'Pago',
      waiting_payment: 'Aguardando',
      refused: 'Recusado',
      refunded: 'Reembolsado',
      chargedback: 'Chargeback',
      pending: 'Pendente',
    }
    return map[status] || status || '—'
  }

  $: rangeHint =
    dateRange?.start && dateRange?.end
      ? `${dateRange.start} → ${dateRange.end}`
      : 'Últimos 30 dias'

  $: salesCount = stats != null ? formatInt(stats.total_sales) : '—'
  $: revenue = stats != null ? formatBRL(stats.total_net_amount) : '—'
  $: balanceValue = balances != null ? formatBRL(balances.available) : '—'
  $: balanceHint =
    balances != null
      ? `Pendente: ${formatBRL(balances.pending)}`
      : errors?.balances || 'Saldo disponível'
  $: eventsCount = formatInt(events?.length ?? 0)

  $: saleRows = (sales || []).map((s) => [
    s.reference || s.id || '—',
    statusLabel(s.status),
    formatBRL(s.net_amount),
    formatDate(s.created_at),
  ])

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
</script>

<svelte:head>
  <title>Dashboard · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-green-950">Dashboard</h1>
    <p class="mt-1 text-sm text-green-800/70">Visão geral das operações Kiwify</p>
    <p class="mt-0.5 text-xs text-green-700/60">Período: {rangeHint}</p>
  </div>

  {#if errorMessages.length > 0}
    <div
      class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900"
      data-testid="dashboard-errors"
      role="status"
    >
      <p class="font-medium">Alguns dados não puderam ser carregados:</p>
      <ul class="mt-1 list-inside list-disc text-amber-800/90">
        {#each errorMessages as msg}
          <li>{msg}</li>
        {/each}
      </ul>
    </div>
  {/if}

  <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
    <StatCard
      title="Vendas (30d)"
      value={salesCount}
      hint={errors?.stats || rangeHint}
    />
    <StatCard
      title="Receita (30d)"
      value={revenue}
      hint={errors?.stats || 'Valor líquido'}
    />
    <StatCard title="Saldo" value={balanceValue} hint={balanceHint} />
    <StatCard
      title="Eventos recentes"
      value={eventsCount}
      hint={errors?.events || 'Webhooks locais'}
    />
  </div>

  <div class="mt-8 grid gap-6 lg:grid-cols-2">
    <section>
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-green-800/80">
        Últimas vendas
      </h2>
      <DataTable
        headers={['Referência', 'Status', 'Valor', 'Data']}
        rows={saleRows}
      >
        <tr>
          <td colspan="4" class="px-4 py-8 text-center text-sm text-green-700/60">
            {#if errors?.sales}
              {errors.sales}
            {:else}
              Nenhuma venda no período.
            {/if}
          </td>
        </tr>
      </DataTable>
    </section>

    <section>
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-green-800/80">
        Eventos recentes
      </h2>
      <div class="rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm">
        {#if events?.length > 0}
          <ul class="divide-y divide-green-50" data-testid="events-list">
            {#each events as ev}
              <li class="flex items-start justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-green-950">
                    {ev.eventType || 'evento'}
                  </p>
                  <p class="text-xs text-green-700/70">{formatDate(ev.receivedAt)}</p>
                </div>
                <span
                  class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium"
                  class:bg-green-100={ev.processedOk}
                  class:text-green-800={ev.processedOk}
                  class:bg-amber-100={!ev.processedOk}
                  class:text-amber-800={!ev.processedOk}
                >
                  {ev.processedOk ? 'OK' : 'Pendente'}
                </span>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="text-sm text-green-700/60">
            {#if errors?.events}
              {errors.events}
            {:else}
              Nenhum evento de webhook recebido ainda.
            {/if}
          </p>
        {/if}
      </div>
    </section>
  </div>
</AppLayout>
