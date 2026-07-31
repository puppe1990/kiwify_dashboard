<script>
  import AppLayout from '../components/AppLayout.svelte'

  export let account = null
  export let errors = {}
  export let site = {}
  export let flash = {}

  $: errorMessages = Object.values(errors || {}).filter(Boolean)

  $: rawEntries = account?.raw
    ? Object.entries(account.raw).filter(([k]) => !['id', 'name', 'email'].includes(k))
    : []

  function formatValue(v) {
    if (v == null) return '—'
    if (typeof v === 'object') {
      try {
        return JSON.stringify(v, null, 2)
      } catch {
        return String(v)
      }
    }
    return String(v)
  }
</script>

<svelte:head>
  <title>Conta · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-green-950">Conta</h1>
    <p class="mt-1 text-sm text-green-800/70">
      Detalhes da conta Kiwify via API Pública
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

  {#if account}
    <div
      class="overflow-hidden rounded-xl border border-[#bbf7d0] bg-white shadow-sm"
      data-testid="account-card"
    >
      <dl class="divide-y divide-green-50">
        <div class="grid gap-1 px-5 py-4 sm:grid-cols-3 sm:gap-4">
          <dt class="text-sm font-medium text-green-900">ID</dt>
          <dd class="text-sm text-green-950 sm:col-span-2 font-mono" data-testid="account-id">
            {account.id || '—'}
          </dd>
        </div>
        <div class="grid gap-1 px-5 py-4 sm:grid-cols-3 sm:gap-4">
          <dt class="text-sm font-medium text-green-900">Nome</dt>
          <dd class="text-sm text-green-950 sm:col-span-2" data-testid="account-name">
            {account.name || '—'}
          </dd>
        </div>
        <div class="grid gap-1 px-5 py-4 sm:grid-cols-3 sm:gap-4">
          <dt class="text-sm font-medium text-green-900">E-mail</dt>
          <dd class="text-sm text-green-950 sm:col-span-2" data-testid="account-email">
            {account.email || '—'}
          </dd>
        </div>
        {#each rawEntries as [key, value]}
          <div class="grid gap-1 px-5 py-4 sm:grid-cols-3 sm:gap-4">
            <dt class="text-sm font-medium text-green-900 break-all">{key}</dt>
            <dd class="text-sm text-green-950 sm:col-span-2">
              {#if typeof value === 'object' && value !== null}
                <pre
                  class="max-h-48 overflow-auto rounded-lg border border-green-100 bg-green-50/40 p-2 font-mono text-[11px] leading-relaxed"
                >{formatValue(value)}</pre>
              {:else}
                <span class="break-all">{formatValue(value)}</span>
              {/if}
            </dd>
          </div>
        {/each}
      </dl>
    </div>
  {:else if !errorMessages.length}
    <div
      class="rounded-xl border border-dashed border-[#bbf7d0] bg-white px-6 py-12 text-center shadow-sm"
    >
      <p class="text-sm text-green-800/70">Nenhum dado de conta disponível.</p>
    </div>
  {/if}
</AppLayout>
