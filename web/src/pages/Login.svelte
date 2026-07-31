<script>
  import { useForm, inertia } from '@inertiajs/svelte'
  import PasswordInput from '../components/PasswordInput.svelte'
  import AuthLayout from '../components/AuthLayout.svelte'

  export let errors = {}
  export let site = {}
  export let flash = {}

  let form = useForm({ email: 'demo@example.com', password: 'password' })

  function submit() {
    form.post('/login')
  }

  const fieldClass =
    'block w-full rounded-lg border border-green-200 bg-white p-2.5 text-sm text-green-950 placeholder:text-green-700/40 focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600'
</script>

<svelte:head>
  <title>Entrar · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AuthLayout
  {site}
  {flash}
  title="Entrar"
  subtitle="Acesse o painel com sua conta local."
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
      <div class="mb-1 flex items-center justify-between">
        <label for="password" class="block text-sm font-medium text-green-900">Senha</label>
        <a
          href="/forgot-password"
          use:inertia
          class="text-xs font-medium text-[#166534] hover:underline"
        >
          Esqueci a senha
        </a>
      </div>
      <PasswordInput
        bind:value={form.password}
        name="password"
        id="password"
        autocomplete="current-password"
        className={fieldClass}
      />
      {#if errors.password}
        <p class="mt-1 text-xs text-red-600">{errors.password}</p>
      {/if}
    </div>

    <button
      type="submit"
      class="w-full rounded-lg bg-[#166534] px-4 py-2.5 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-60"
      disabled={form.processing}
    >
      {form.processing ? 'Entrando…' : 'Entrar'}
    </button>
  </form>

  <p class="mt-6 text-center text-sm text-green-800/70">
    Não tem conta?
    <a href="/signup" use:inertia class="font-medium text-[#166534] hover:underline">Criar conta</a>
  </p>

  <p class="mt-4 rounded-lg border border-green-100 bg-green-50/60 px-3 py-2 text-center text-[11px] text-green-800/70">
    Dev: <span class="font-mono">demo@example.com</span> / <span class="font-mono">password</span>
  </p>
</AuthLayout>
