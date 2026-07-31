<script>
  import { inertia } from '@inertiajs/svelte'

  export let title = 'Home'
  export let site = {}
  export let flash = {}
  export let labels = {}
</script>

<svelte:head>
  <title>{title || 'Kiwify Ops'} · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<div class="min-h-screen bg-[#f0fdf4] text-green-950">
  <header class="border-b border-green-200/60 bg-white/80 backdrop-blur">
    <div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
      <span class="text-lg font-semibold text-[#14532d]">{site.appName || 'Kiwify Ops'}</span>
      <nav class="flex items-center gap-3 text-sm">
        <a href="/login" use:inertia class="font-medium text-[#166534] hover:underline">
          {labels.login || 'Entrar'}
        </a>
        <a
          href="/login"
          use:inertia
          class="rounded-lg bg-[#166534] px-3 py-1.5 font-medium text-white hover:bg-[#14532d]"
        >
          {labels.dashboard || 'Dashboard'}
        </a>
      </nav>
    </div>
  </header>

  <main class="mx-auto flex max-w-3xl flex-col items-center px-6 py-20 text-center">
    {#if flash?.notice || flash?.success}
      <p class="mb-6 rounded-lg border border-green-200 bg-white px-4 py-2 text-sm text-green-900">
        {flash.notice || flash.success}
      </p>
    {/if}

    <p class="text-xs font-semibold uppercase tracking-wider text-[#22c55e]">Operações Kiwify</p>
    <h1 class="mt-3 text-4xl font-semibold tracking-tight text-green-950 md:text-5xl">
      {labels.heading || 'Painel de operações Kiwify'}
    </h1>
    <p data-testid="inertia-ready" class="mt-4 max-w-xl text-lg text-green-800/80">
      {labels.subtitle ||
        'Proxy ao vivo da API Pública: vendas, financeiro, afiliados, webhooks e auditoria.'}
    </p>
    <p class="mt-3 text-sm text-green-700/70">{labels.stack || 'Go · Inertia · Svelte · SQLite'}</p>

    <div class="mt-10 flex flex-wrap items-center justify-center gap-3">
      <a
        href="/login"
        use:inertia
        class="rounded-lg bg-[#166534] px-5 py-2.5 text-sm font-medium text-white hover:bg-[#14532d]"
      >
        {labels.login || 'Entrar'}
      </a>
      <a
        href="/signup"
        use:inertia
        class="rounded-lg border border-green-300 bg-white px-5 py-2.5 text-sm font-medium text-green-900 hover:bg-green-50"
      >
        Criar conta
      </a>
    </div>
  </main>
</div>
