<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let affiliates = []
  export let pagination = {}
  export let filters = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

  let status = filters?.status || ''
  let search = filters?.search || ''

  $: status = filters?.status ?? status
  $: search = filters?.search ?? search

  /** API amounts are in centavos (minor units). */
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

  function statusLabel(s) {
    const map = {
      active: 'Ativo',
      blocked: 'Bloqueado',
      refused: 'Recusado',
      pending: 'Pendente',
    }
    return map[s] || s || '—'
  }

  function applyFilters() {
    router.get(
      '/affiliates',
      {
        status: status || undefined,
        search: search || undefined,
      },
      { preserveState: true, replace: true },
    )
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: totalCount = pagination?.count ?? affiliates?.length ?? 0
</script>

<svelte:head>
  <title>Afiliados · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-green-950">Afiliados</h1>
      <p class="mt-1 text-sm text-green-800/70">
        Listagem e edição de afiliados via API Kiwify
      </p>
    </div>
    <p class="text-xs text-green-700/70">{totalCount} registro(s)</p>
  </div>

  <form
    class="mb-6 flex flex-wrap items-end gap-3 rounded-xl border border-[#bbf7d0] bg-white p-4 shadow-sm"
    on:submit|preventDefault={applyFilters}
    data-testid="affiliates-filters"
  >
    <div>
      <label for="search" class="mb-1 block text-xs font-medium text-green-900">Busca</label>
      <input
        id="search"
        type="search"
        bind:value={search}
        placeholder="Nome ou e-mail"
        class="rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
      />
    </div>
    <div>
      <label for="status" class="mb-1 block text-xs font-medium text-green-900">Status</label>
      <select
        id="status"
        bind:value={status}
        class="rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
      >
        <option value="">Todos</option>
        <option value="active">Ativo</option>
        <option value="blocked">Bloqueado</option>
        <option value="refused">Recusado</option>
      </select>
    </div>
    <button
      type="submit"
      class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d]"
    >
      Filtrar
    </button>
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

  <DataTable headers={['Nome', 'E-mail', 'Status', 'Comissão', 'Produto', 'Criado em', '']}>
    {#if !affiliates || affiliates.length === 0}
      <tr>
        <td colspan="7" class="px-4 py-8 text-center text-sm text-green-700/60">
          {#if errors?.list || errors?.client}
            {errors.list || errors.client}
          {:else}
            Nenhum afiliado encontrado.
          {/if}
        </td>
      </tr>
    {:else}
      {#each affiliates as a}
        <tr class="hover:bg-green-50/40">
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            <a
              href={`/affiliates/${a.affiliate_id}`}
              use:inertia
              class="font-medium text-[#166534] hover:underline"
            >
              {a.name || a.affiliate_id || '—'}
            </a>
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {a.email || '—'}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {statusLabel(a.status)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatBRL(a.commission)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {a.product?.name || '—'}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatDate(a.created_at)}
          </td>
          <td class="px-4 py-3 text-sm whitespace-nowrap">
            <a
              href={`/affiliates/${a.affiliate_id}`}
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
