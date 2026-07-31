<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let sales = []
  export let dateRange = {}
  export let pagination = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

  let startDate = dateRange?.start || ''
  let endDate = dateRange?.end || ''

  $: startDate = dateRange?.start || startDate
  $: endDate = dateRange?.end || endDate

  function formatBRL(amount) {
    if (amount == null || amount === '' || Number.isNaN(Number(amount))) return '—'
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(Number(amount) / 100)
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

  function applyFilters() {
    router.get(
      '/sales',
      { start_date: startDate, end_date: endDate },
      { preserveState: true, replace: true },
    )
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: totalCount = pagination?.count ?? sales?.length ?? 0
</script>

<svelte:head>
  <title>Vendas · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-green-950">Vendas</h1>
      <p class="mt-1 text-sm text-green-800/70">
        Listagem da API Kiwify (máx. 90 dias por consulta)
      </p>
    </div>
  </div>

  <form
    class="mb-6 flex flex-wrap items-end gap-3 rounded-xl border border-[#bbf7d0] bg-white p-4 shadow-sm"
    on:submit|preventDefault={applyFilters}
    data-testid="sales-filters"
  >
    <div>
      <label for="start_date" class="mb-1 block text-xs font-medium text-green-900">Data início</label>
      <input
        id="start_date"
        type="date"
        bind:value={startDate}
        class="rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
      />
    </div>
    <div>
      <label for="end_date" class="mb-1 block text-xs font-medium text-green-900">Data fim</label>
      <input
        id="end_date"
        type="date"
        bind:value={endDate}
        class="rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
      />
    </div>
    <button
      type="submit"
      class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d]"
    >
      Filtrar
    </button>
    <p class="w-full text-xs text-green-700/70 sm:w-auto sm:ml-auto">
      {#if dateRange?.start && dateRange?.end}
        Período efetivo: {dateRange.start} → {dateRange.end}
        {#if totalCount !== undefined && totalCount !== null}
          · {totalCount} registro(s)
        {/if}
      {/if}
    </p>
  </form>

  {#if errorMessages.length > 0}
    <div
      class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900"
      role="status"
    >
      <ul class="list-inside list-disc">
        {#each errorMessages as msg}
          <li>{msg}</li>
        {/each}
      </ul>
    </div>
  {/if}

  <DataTable headers={['Referência', 'Status', 'Cliente', 'Produto', 'Valor', 'Data', '']}>
    {#if !sales || sales.length === 0}
      <tr>
        <td colspan="7" class="px-4 py-8 text-center text-sm text-green-700/60">
          {#if errors?.list || errors?.client}
            {errors.list || errors.client}
          {:else}
            Nenhuma venda no período.
          {/if}
        </td>
      </tr>
    {:else}
      {#each sales as s}
        <tr class="hover:bg-green-50/40">
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            <a
              href={`/sales/${s.id}`}
              use:inertia
              class="font-medium text-[#166534] hover:underline"
            >
              {s.reference || s.id || '—'}
            </a>
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">{statusLabel(s.status)}</td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {s.customer?.name || s.customer?.email || '—'}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {s.product?.name || '—'}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">{formatBRL(s.net_amount)}</td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">{formatDate(s.created_at)}</td>
          <td class="px-4 py-3 text-sm whitespace-nowrap">
            <a
              href={`/sales/${s.id}`}
              use:inertia
              class="text-[#166534] hover:underline"
            >
              Detalhes
            </a>
          </td>
        </tr>
      {/each}
    {/if}
  </DataTable>
</AppLayout>
