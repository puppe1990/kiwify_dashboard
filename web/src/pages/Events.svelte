<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'

  export let events = []
  export let pagination = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

  /** @type {Record<string | number, boolean>} */
  let expanded = {}

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

  function prettyJSON(raw) {
    if (!raw) return ''
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }

  function toggle(id) {
    expanded = { ...expanded, [id]: !expanded[id] }
  }

  function goPage(page) {
    router.get('/events', { page }, { preserveState: true, replace: true })
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: page = pagination?.page ?? 1
  $: hasMore = !!pagination?.has_more
</script>

<svelte:head>
  <title>Eventos · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-green-950">Eventos</h1>
      <p class="mt-1 text-sm text-green-800/70">
        Feed local de webhooks recebidos em
        <span class="font-mono text-xs">/webhooks/kiwify/…</span>
      </p>
    </div>
    <a
      href="/webhooks"
      use:inertia
      class="text-sm font-medium text-[#166534] hover:underline"
    >
      Gerenciar webhooks →
    </a>
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

  {#if events?.length}
    <ul class="space-y-3" data-testid="events-feed">
      {#each events as ev}
        <li
          class="overflow-hidden rounded-xl border border-[#bbf7d0] bg-white shadow-sm"
          data-testid="event-row"
        >
          <button
            type="button"
            class="flex w-full items-start justify-between gap-3 px-4 py-3 text-left hover:bg-green-50/40"
            on:click={() => toggle(ev.id)}
            aria-expanded={!!expanded[ev.id]}
          >
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold text-green-950">
                {ev.eventType || 'unknown'}
              </p>
              <p class="text-xs text-green-700/70">{formatDate(ev.receivedAt)}</p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span
                class="rounded-full px-2 py-0.5 text-xs font-medium"
                class:bg-green-100={ev.processedOk}
                class:text-green-800={ev.processedOk}
                class:bg-amber-100={!ev.processedOk}
                class:text-amber-800={!ev.processedOk}
              >
                {ev.processedOk ? 'OK' : 'Pendente'}
              </span>
              <span class="text-xs text-green-700/60">{expanded[ev.id] ? '▲' : '▼'}</span>
            </div>
          </button>

          {#if expanded[ev.id]}
            <div class="border-t border-green-100 bg-green-50/30 px-4 py-3" data-testid="event-payload">
              <p class="mb-1 text-xs font-medium text-green-900">Payload</p>
              <pre
                class="max-h-80 overflow-auto rounded-lg border border-green-100 bg-white p-3 font-mono text-[11px] leading-relaxed text-green-950"
              >{prettyJSON(ev.payloadJson)}</pre>
              {#if ev.headersJson}
                <p class="mb-1 mt-3 text-xs font-medium text-green-900">Headers</p>
                <pre
                  class="max-h-40 overflow-auto rounded-lg border border-green-100 bg-white p-3 font-mono text-[11px] leading-relaxed text-green-950"
                >{prettyJSON(ev.headersJson)}</pre>
              {/if}
            </div>
          {/if}
        </li>
      {/each}
    </ul>

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
    >
      <p class="text-sm text-green-800/70">Nenhum evento de webhook recebido ainda.</p>
      <p class="mt-2 text-xs text-green-700/60">
        Configure a URL pública nas Configurações e registre um webhook na Kiwify.
      </p>
    </div>
  {/if}
</AppLayout>
