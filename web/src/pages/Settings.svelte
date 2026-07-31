<script>
  import { useForm, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'

  export let accountId = ''
  export let clientId = ''
  export let hasSecret = false
  export let webhookReceiveURL = ''
  export let apiStatus = 'untested'
  export let apiStatusMessage = ''
  export let errors = {}
  export let site = {}
  export let flash = {}

  let form = useForm({
    account_id: accountId || '',
    client_id: clientId || '',
    client_secret: '',
  })

  let testing = false
  let copied = false

  function submit() {
    form.post('/settings')
  }

  function testConnection() {
    testing = true
    router.post(
      '/settings',
      { intent: 'test', test_only: '1' },
      {
        onFinish: () => {
          testing = false
        },
      },
    )
  }

  async function copyWebhook() {
    if (!webhookReceiveURL) return
    try {
      await navigator.clipboard.writeText(webhookReceiveURL)
      copied = true
      setTimeout(() => {
        copied = false
      }, 2000)
    } catch {
      // ignore clipboard errors
    }
  }

  $: statusStyles = {
    ok: 'border-green-300 bg-green-50 text-green-900',
    untested: 'border-amber-300 bg-amber-50 text-amber-950',
    expired: 'border-amber-300 bg-amber-50 text-amber-950',
    missing: 'border-red-300 bg-red-50 text-red-900',
    error: 'border-red-300 bg-red-50 text-red-900',
  }
  $: statusLabels = {
    ok: 'API conectada',
    untested: 'Ainda não validado',
    expired: 'Token expirado',
    missing: 'Credenciais incompletas',
    error: 'Falha na conexão',
  }
</script>

<svelte:head>
  <title>Configurações · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mx-auto max-w-lg space-y-4">
    <div
      class="rounded-xl border px-4 py-3 text-sm {statusStyles[apiStatus] || statusStyles.untested}"
      data-testid="api-status"
      role="status"
    >
      <p class="font-semibold">{statusLabels[apiStatus] || 'Status da API'}</p>
      {#if apiStatusMessage}
        <p class="mt-1 opacity-90">{apiStatusMessage}</p>
      {/if}
    </div>

    <div class="rounded-xl border border-[#bbf7d0] bg-white p-6 shadow-sm">
      <h1 class="mb-1 text-2xl font-semibold text-green-900">Configurações</h1>
      <p class="mb-6 text-sm text-green-800/80">Credenciais da API Pública da Kiwify</p>

      <form on:submit|preventDefault={submit} class="space-y-4">
        <div>
          <label for="account_id" class="mb-1 block text-sm font-medium text-green-900">Account ID</label>
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
          <label for="client_id" class="mb-1 block text-sm font-medium text-green-900">Client ID</label>
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
          <label for="client_secret" class="mb-1 block text-sm font-medium text-green-900">Client Secret</label>
          <input
            id="client_secret"
            type="password"
            bind:value={form.client_secret}
            placeholder={hasSecret ? '•••••••• (deixe em branco para manter)' : ''}
            autocomplete="new-password"
            class="block w-full rounded-lg border border-green-200 p-2.5 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
          />
          {#if hasSecret}
            <p class="mt-1 text-xs text-green-700">
              Já existe um secret salvo. Preencha apenas se quiser substituí-lo.
            </p>
          {/if}
          {#if errors.client_secret}<p class="mt-1 text-xs text-red-600">{errors.client_secret}</p>{/if}
        </div>

        <div>
          <label for="webhook_url" class="mb-1 block text-sm font-medium text-green-900"
            >URL de recebimento de webhooks</label
          >
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
              {copied ? 'Copiado!' : 'Copiar'}
            </button>
          </div>
        </div>

        <div class="flex flex-col gap-2 sm:flex-row">
          <button
            type="submit"
            class="flex-1 rounded-lg bg-[#166534] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-60"
            disabled={form.processing || testing}
          >
            {form.processing ? 'Salvando e testando…' : 'Salvar e validar'}
          </button>
          <button
            type="button"
            on:click={testConnection}
            class="flex-1 rounded-lg border border-green-300 bg-white px-4 py-2.5 text-sm font-medium text-green-900 hover:bg-green-50 disabled:opacity-60"
            disabled={form.processing || testing || !hasSecret}
            data-testid="test-connection"
          >
            {testing ? 'Testando…' : 'Testar conexão'}
          </button>
        </div>
      </form>
    </div>
  </div>
</AppLayout>
