<script>
  import { useForm, inertia } from '@inertiajs/svelte'
  import PasswordInput from '../components/PasswordInput.svelte'
  import AuthLayout from '../components/AuthLayout.svelte'

  export let errors = {}
  export let site = {}
  export let flash = {}

  let form = useForm({ email: '', password: '', password_confirmation: '' })

  function submit() {
    form.post('/signup')
  }

  const fieldClass =
    'block w-full rounded-lg border border-green-200 bg-white p-2.5 text-sm text-green-950 placeholder:text-green-700/40 focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600'
</script>

<svelte:head>
  <title>Criar conta · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AuthLayout
  {site}
  {flash}
  title="Criar conta"
  subtitle="Cadastro local para acessar o painel."
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

    <div>
      <label for="password" class="mb-1 block text-sm font-medium text-green-900">Senha</label>
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
      {form.processing ? 'Criando…' : 'Criar conta'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-green-800/70">
    Já tem conta?
    <a href="/login" use:inertia class="font-medium text-[#166534] hover:underline">Entrar</a>
  </p>
</AuthLayout>
