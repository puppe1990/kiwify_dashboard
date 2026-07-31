<script>
  import { inertia } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let webhooks = []
  export let pagination = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

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

  function triggersLabel(triggers) {
    if (!triggers || !triggers.length) return '—'
    if (triggers.length <= 2) return triggers.join(', ')
    return `${triggers.slice(0, 2).join(', ')} +${triggers.length - 2}`
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: totalCount = pagination?.count ?? webhooks?.length ?? 0
</script>

<svelte:head>
  <title>Webhooks · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-green-950">Webhooks</h1>
      <p class="mt-1 text-sm text-green-800/70">
        Cadastro de webhooks na API Kiwify (URL de destino + gatilhos)
      </p>
    </div>
    <div class="flex items-center gap-3">
      <p class="text-xs text-green-700/70">{totalCount} registro(s)</p>
      <a
        href="/webhooks/new"
        use:inertia
        class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d]"
        data-testid="webhook-new"
      >
        Novo webhook
      </a>
    </div>
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

  {#if webhooks?.length}
    <DataTable headers={['Nome', 'URL', 'Produtos', 'Gatilhos', 'Criado', '']}>
      {#each webhooks as wh}
        <tr class="hover:bg-green-50/40" data-testid="webhook-row">
          <td class="px-4 py-3 text-green-950/90">
            <a
              href={`/webhooks/${wh.id}`}
              use:inertia
              class="font-medium text-[#166534] hover:underline"
            >
              {wh.name || wh.id}
            </a>
          </td>
          <td class="max-w-xs truncate px-4 py-3 text-xs text-green-900/80" title={wh.url}>
            {wh.url || '—'}
          </td>
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap">
            {wh.products === 'all' ? 'Todos' : wh.products || '—'}
          </td>
          <td class="px-4 py-3 text-xs text-green-900/80">
            {triggersLabel(wh.triggers)}
          </td>
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap">
            {formatDate(wh.created_at)}
          </td>
          <td class="px-4 py-3 text-right">
            <a
              href={`/webhooks/${wh.id}`}
              use:inertia
              class="text-sm font-medium text-[#166534] hover:underline"
            >
              Editar
            </a>
          </td>
        </tr>
      {/each}
    </DataTable>
  {:else if !errorMessages.length}
    <div
      class="rounded-xl border border-dashed border-[#bbf7d0] bg-white px-6 py-12 text-center shadow-sm"
    >
      <p class="text-sm text-green-800/70">Nenhum webhook cadastrado.</p>
      <a
        href="/webhooks/new"
        use:inertia
        class="mt-4 inline-block rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d]"
      >
        Criar primeiro webhook
      </a>
    </div>
  {/if}
</AppLayout>
