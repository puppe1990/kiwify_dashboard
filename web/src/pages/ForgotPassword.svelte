<script>
  import { useForm, inertia } from '@inertiajs/svelte'
  import AuthLayout from '../components/AuthLayout.svelte'

  export let errors = {}
  export let site = {}
  export let flash = {}

  let form = useForm({ email: '' })

  function submit() {
    form.post('/forgot-password')
  }

  const fieldClass =
    'block w-full rounded-lg border border-green-200 bg-white p-2.5 text-sm text-green-950 placeholder:text-green-700/40 focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600'
</script>

<svelte:head>
  <title>Esqueci a senha · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AuthLayout
  {site}
  {flash}
  title="Esqueci a senha"
  subtitle="Enviaremos um link de redefinição se o e-mail existir."
>
  <form on:submit|preventDefault={submit} class="space-y-4">
    <div>
      <label for="email" class="mb-1 block text-sm font-medium text-green-900">E-mail</label>
      <input
        id="email"
        type="email"
        name="email"
        autocomplete="username"
        bind:value={form.email}
        class={fieldClass}
      />
      {#if errors.email}
        <p class="mt-1 text-xs text-red-600">{errors.email}</p>
      {/if}
    </div>

    <button
      type="submit"
      class="w-full rounded-lg bg-[#166534] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-60"
      disabled={form.processing}
    >
      {form.processing ? 'Enviando…' : 'Enviar link'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-green-800/70">
    <a href="/login" use:inertia class="font-medium text-[#166534] hover:underline"
      >← Voltar ao login</a
    >
  </p>
</AuthLayout>
