<script>
  import { useForm, inertia } from '@inertiajs/svelte'
  import PasswordInput from '../components/PasswordInput.svelte'
  import AuthLayout from '../components/AuthLayout.svelte'

  export let errors = {}
  export let token = ''
  export let site = {}
  export let flash = {}

  let form = useForm({ token, password: '', password_confirmation: '' })

  function submit() {
    form.post('/reset-password')
  }

  const fieldClass =
    'block w-full rounded-lg border border-green-200 bg-white p-2.5 text-sm text-green-950 placeholder:text-green-700/40 focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600'
</script>

<svelte:head>
  <title>Nova senha · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AuthLayout
  {site}
  {flash}
  title="Redefinir senha"
  subtitle="Escolha uma nova senha para sua conta."
>
  {#if errors.token}
    <p class="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
      {errors.token}
    </p>
  {/if}

  <form on:submit|preventDefault={submit} class="space-y-4">
    <input type="hidden" bind:value={form.token} />

    <div>
      <label for="password" class="mb-1 block text-sm font-medium text-green-900">Nova senha</label>
      <PasswordInput
        bind:value={form.password}
        name="password"
        id="password"
        autocomplete="new-password"
        className={fieldClass}
      />
      {#if errors.password}
        <p class="mt-1 text-xs text-red-600">{errors.password}</p>
      {/if}
    </div>

    <div>
      <label for="password_confirmation" class="mb-1 block text-sm font-medium text-green-900"
        >Confirmar senha</label
      >
      <PasswordInput
        bind:value={form.password_confirmation}
        name="password_confirmation"
        id="password_confirmation"
        autocomplete="new-password"
        className={fieldClass}
      />
      {#if errors.password_confirmation}
        <p class="mt-1 text-xs text-red-600">{errors.password_confirmation}</p>
      {/if}
    </div>

    <button
      type="submit"
      class="w-full rounded-lg bg-[#166534] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-60"
      disabled={form.processing}
    >
      {form.processing ? 'Salvando…' : 'Redefinir senha'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-green-800/70">
    <a href="/login" use:inertia class="font-medium text-[#166534] hover:underline"
      >← Voltar ao login</a
    >
  </p>
</AuthLayout>
