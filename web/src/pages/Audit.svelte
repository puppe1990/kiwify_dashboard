<script>
  import { router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import DataTable from '../components/DataTable.svelte'

  export let logs = []
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
      second: '2-digit',
    })
  }

  function statusTone(status) {
    const n = Number(status)
    if (n >= 200 && n < 300) return 'ok'
    if (n >= 400) return 'err'
    return 'other'
  }

  function resourceLabel(log) {
    if (!log) return '—'
    const type = log.resourceType || ''
    const id = log.resourceId || ''
    if (type && id) return `${type}:${id}`
    return type || id || '—'
  }

  function summarySnippet(raw) {
    if (!raw) return '—'
    const s = String(raw)
    return s.length > 80 ? s.slice(0, 80) + '…' : s
  }

  function goPage(page) {
    router.get('/audit', { page }, { preserveState: true, replace: true })
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: page = pagination?.page ?? 1
  $: hasMore = !!pagination?.has_more

  $: tableHeaders = ['Ação', 'Recurso', 'Status', 'Horário', 'Usuário', 'Resumo']
</script>

<svelte:head>
  <title>Auditoria · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-green-950">Auditoria</h1>
    <p class="mt-1 text-sm text-green-800/70">
      Histórico de ações sensíveis (reembolsos, saques, afiliados, webhooks)
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

  {#if logs?.length}
    <DataTable headers={tableHeaders}>
      {#each logs as log}
        <tr class="hover:bg-green-50/40" data-testid="audit-row">
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap font-medium">
            {log.action || '—'}
          </td>
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap font-mono text-xs">
            {resourceLabel(log)}
          </td>
          <td class="px-4 py-3 whitespace-nowrap">
            <span
              class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
              class:bg-green-100={statusTone(log.responseStatus) === 'ok'}
              class:text-green-800={statusTone(log.responseStatus) === 'ok'}
              class:bg-red-100={statusTone(log.responseStatus) === 'err'}
              class:text-red-800={statusTone(log.responseStatus) === 'err'}
              class:bg-gray-100={statusTone(log.responseStatus) === 'other'}
              class:text-gray-800={statusTone(log.responseStatus) === 'other'}
            >
              {log.responseStatus ?? '—'}
            </span>
          </td>
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap text-xs">
            {formatDate(log.createdAt)}
          </td>
          <td class="px-4 py-3 text-green-950/90 whitespace-nowrap">
            #{log.userId ?? '—'}
          </td>
          <td
            class="px-4 py-3 text-green-950/80 max-w-xs truncate text-xs font-mono"
            title={log.requestSummary || ''}
          >
            {summarySnippet(log.requestSummary)}
          </td>
        </tr>
      {/each}
    </DataTable>

    <div class="mt-6 flex items-center justify-between gap-3">
      <button
        type="button"
        class="rounded-lg border border-green-200 px-3 py-1.5 text-sm text-green-900 hover:bg-green-50 disabled:opacity-40"
        disabled={page <= 1}
        on:click={() => goPage(page - 1)}
      >
        ← Anterior
      </button>
      <span class="text-xs text-green-700/70">Página {page}</span>
      <button
        type="button"
        class="rounded-lg border border-green-200 px-3 py-1.5 text-sm text-green-900 hover:bg-green-50 disabled:opacity-40"
        disabled={!hasMore}
        on:click={() => goPage(page + 1)}
      >
        Próxima →
      </button>
    </div>
  {:else if !errorMessages.length}
    <div
      class="rounded-xl border border-dashed border-[#bbf7d0] bg-white px-6 py-12 text-center shadow-sm"
      data-testid="audit-empty"
    >
      <p class="text-sm text-green-800/70">Nenhuma ação sensível registrada ainda.</p>
      <p class="mt-2 text-xs text-green-700/60">
        Reembolsos, saques, edições de afiliados e alterações de webhooks aparecem aqui.
      </p>
    </div>
  {/if}
</AppLayout>
