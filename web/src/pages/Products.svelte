<script>
  import { inertia } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let products = []
  export let pagination = {}
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
      active: 'Ativo',
      inactive: 'Inativo',
      draft: 'Rascunho',
      archived: 'Arquivado',
    }
    return map[status] || status || '—'
  }

  function typeLabel(type) {
    const map = {
      membership: 'Área de membros',
      course: 'Curso',
      ebook: 'E-book',
      physical: 'Físico',
      service: 'Serviço',
    }
    return map[type] || type || '—'
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: totalCount = pagination?.count ?? products?.length ?? 0
</script>

<svelte:head>
  <title>Produtos · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-green-950">Produtos</h1>
      <p class="mt-1 text-sm text-green-800/70">
        Listagem somente leitura da API Kiwify
      </p>
      <p class="mt-1 text-xs text-green-700/70">
        A criação e edição de produtos é feita apenas no dashboard da Kiwify.
        Esta API é somente leitura.
      </p>
    </div>
    <p class="text-xs text-green-700/70" data-testid="products-count">
      {totalCount} registro(s)
    </p>
  </div>

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

  <DataTable headers={['Nome', 'Status', 'Tipo', 'Preço', 'Criado em', '']}>
    {#if !products || products.length === 0}
      <tr>
        <td colspan="6" class="px-4 py-8 text-center text-sm text-green-700/60">
          {#if errors?.list || errors?.client}
            {errors.list || errors.client}
          {:else}
            Nenhum produto encontrado.
          {/if}
        </td>
      </tr>
    {:else}
      {#each products as p}
        <tr class="hover:bg-green-50/40">
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            <a
              href={`/products/${p.id}`}
              use:inertia
              class="font-medium text-[#166534] hover:underline"
            >
              {p.name || p.id || '—'}
            </a>
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {statusLabel(p.status)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {typeLabel(p.type)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatBRL(p.price)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatDate(p.created_at)}
          </td>
          <td class="px-4 py-3 text-sm whitespace-nowrap">
            <a
              href={`/products/${p.id}`}
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
