<script>
  import { inertia } from '@inertiajs/svelte'
  import FlashBanner from './FlashBanner.svelte'

  export let site = {}
  export let flash = {}
  export let title = ''
  export let subtitle = ''
</script>

<div class="min-h-screen bg-[#f0fdf4] text-green-950">
  <div class="flex min-h-screen flex-col lg:flex-row">
    <!-- Brand panel -->
    <aside
      class="relative flex flex-col justify-between overflow-hidden bg-[#14532d] px-8 py-10 text-green-50 lg:w-[42%] lg:px-12 lg:py-14"
    >
      <div
        class="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-[#22c55e]/20 blur-2xl"
      ></div>
      <div
        class="pointer-events-none absolute -bottom-20 -left-10 h-64 w-64 rounded-full bg-[#166534]/80 blur-3xl"
      ></div>

      <div class="relative z-10">
        <a href="/" use:inertia class="inline-flex items-center gap-2">
          <span
            class="flex h-9 w-9 items-center justify-center rounded-xl bg-[#22c55e] text-sm font-bold text-white shadow-sm"
          >
            K
          </span>
          <span class="text-lg font-semibold tracking-tight text-white">
            {site.appName || 'Kiwify Ops'}
          </span>
        </a>
        <p class="mt-1 text-xs font-medium uppercase tracking-wider text-green-300/90">
          Operações
        </p>
      </div>

      <div class="relative z-10 mt-12 max-w-sm lg:mt-0">
        <p class="text-xs font-semibold uppercase tracking-wider text-[#86efac]">
          Public API
        </p>
        <h1 class="mt-3 text-3xl font-semibold leading-tight tracking-tight text-white md:text-4xl">
          Painel de operações Kiwify
        </h1>
        <p class="mt-4 text-sm leading-relaxed text-green-100/85">
          Proxy ao vivo da API: vendas, financeiro, afiliados, webhooks e auditoria — com
          confirmação e log nas ações sensíveis.
        </p>
        <ul class="mt-8 space-y-2 text-sm text-green-100/80">
          <li class="flex items-center gap-2">
            <span class="h-1.5 w-1.5 rounded-full bg-[#22c55e]"></span>
            Credenciais cifradas no SQLite
          </li>
          <li class="flex items-center gap-2">
            <span class="h-1.5 w-1.5 rounded-full bg-[#22c55e]"></span>
            OAuth com cache de token
          </li>
          <li class="flex items-center gap-2">
            <span class="h-1.5 w-1.5 rounded-full bg-[#22c55e]"></span>
            Receptor de webhooks com token
          </li>
        </ul>
      </div>

      <p class="relative z-10 mt-12 text-xs text-green-300/70 lg:mt-0">
        Go · Inertia · Svelte · SQLite
      </p>
    </aside>

    <!-- Form panel -->
    <main class="flex flex-1 items-center justify-center px-6 py-12 lg:px-12">
      <div class="w-full max-w-md">
        <FlashBanner {flash} />
        {#if title}
          <h2 class="text-2xl font-semibold tracking-tight text-green-950">{title}</h2>
        {/if}
        {#if subtitle}
          <p class="mt-1.5 text-sm text-green-800/75">{subtitle}</p>
        {/if}
        <div class="mt-8 rounded-2xl border border-[#bbf7d0] bg-white p-6 shadow-sm sm:p-8">
          <slot />
        </div>
      </div>
    </main>
  </div>
</div>
