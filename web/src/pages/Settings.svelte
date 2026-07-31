<script>
  import { useForm } from '@inertiajs/svelte'
  export let accountId = ''
  export let clientId = ''
  export let hasSecret = false
  export let webhookReceiveURL = ''
  export let errors = {}
  export let site = {}
  export let flash = {}

  let form = useForm({
    account_id: accountId || '',
    client_id: clientId || '',
    client_secret: '',
  })

  function submit() {
    form.post('/settings')
  }

  async function copyWebhook() {
    if (!webhookReceiveURL) return
    try {
      await navigator.clipboard.writeText(webhookReceiveURL)
    } catch {
      // ignore clipboard errors
    }
  }
</script>

<svelte:head>
  <title>Configurações · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<div class="min-h-screen bg-green-50 p-6">
  <div class="mx-auto max-w-lg bg-white border border-green-200 rounded-xl shadow-sm p-6">
    <h1 class="text-2xl font-semibold text-green-900 mb-1">Configurações</h1>
    <p class="text-sm text-green-800/80 mb-6">Credenciais da API Pública da Kiwify</p>

    {#if flash.success}
      <p class="mb-4 rounded-lg bg-green-50 px-3 py-2 text-sm text-green-800" data-testid="flash-success">{flash.success}</p>
    {/if}
    {#if flash.warning}
      <p class="mb-4 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">{flash.warning}</p>
    {/if}

    <form on:submit|preventDefault={submit} class="space-y-4">
      <div>
        <label for="account_id" class="block text-sm font-medium text-green-900 mb-1">Account ID</label>
        <input
          id="account_id"
          type="text"
          bind:value={form.account_id}
          autocomplete="off"
          class="block w-full rounded-lg border border-green-200 p-2.5 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
        {#if errors.account_id}<p class="mt-1 text-xs text-red-600">{errors.account_id}</p>{/if}
      </div>

      <div>
        <label for="client_id" class="block text-sm font-medium text-green-900 mb-1">Client ID</label>
        <input
          id="client_id"
          type="text"
          bind:value={form.client_id}
          autocomplete="off"
          class="block w-full rounded-lg border border-green-200 p-2.5 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
        {#if errors.client_id}<p class="mt-1 text-xs text-red-600">{errors.client_id}</p>{/if}
      </div>

      <div>
        <label for="client_secret" class="block text-sm font-medium text-green-900 mb-1">Client Secret</label>
        <input
          id="client_secret"
          type="password"
          bind:value={form.client_secret}
          placeholder={hasSecret ? '•••••••• (deixe em branco para manter)' : ''}
          autocomplete="new-password"
          class="block w-full rounded-lg border border-green-200 p-2.5 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
        {#if hasSecret}
          <p class="mt-1 text-xs text-green-700">Já existe um secret salvo. Preencha apenas se quiser substituí-lo.</p>
        {/if}
        {#if errors.client_secret}<p class="mt-1 text-xs text-red-600">{errors.client_secret}</p>{/if}
      </div>

      <div>
        <label for="webhook_url" class="block text-sm font-medium text-green-900 mb-1">URL de recebimento de webhooks</label>
        <div class="flex gap-2">
          <input
            id="webhook_url"
            type="text"
            readonly
            value={webhookReceiveURL}
            class="block w-full rounded-lg border border-green-200 bg-green-50/50 p-2.5 text-sm text-green-900"
          />
          <button
            type="button"
            on:click={copyWebhook}
            class="shrink-0 rounded-lg border border-green-300 px-3 py-2 text-sm text-green-900 hover:bg-green-50"
          >
            Copiar
          </button>
        </div>
      </div>

      <button
        type="submit"
        class="w-full rounded-lg bg-green-800 px-4 py-2.5 text-sm font-medium text-white hover:bg-green-900 disabled:opacity-60"
        disabled={form.processing}
      >
        {form.processing ? 'Salvando…' : 'Salvar alterações'}
      </button>
    </form>
  </div>
</div>
