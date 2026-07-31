<script>
  import { useForm } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'

  export let errors = {}
  export let site = {}
  export let flash = {}
  let form = useForm({ client_id: '', client_secret: '', account_id: '' })
  function submit() {
    form.post('/setup')
  }
</script>

<svelte:head>
  <title>Configuração inicial · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mx-auto max-w-md rounded-xl border border-[#bbf7d0] bg-white p-6 shadow-sm">
    <h1 class="text-2xl font-semibold text-green-900 mb-1">Configuração Kiwify</h1>
    <p class="text-sm text-green-800/80 mb-6">
      Informe as credenciais da API Pública da Kiwify para começar.
    </p>

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
          autocomplete="new-password"
          class="block w-full rounded-lg border border-green-200 p-2.5 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
        {#if errors.client_secret}<p class="mt-1 text-xs text-red-600">{errors.client_secret}</p>{/if}
      </div>

      <button
        type="submit"
        class="w-full rounded-lg bg-[#166534] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-60"
        disabled={form.processing}
      >
        {form.processing ? 'Salvando…' : 'Salvar e continuar'}
      </button>
    </form>
  </div>
</AppLayout>
