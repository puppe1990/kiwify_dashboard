<script>
  import { inertia, page, router } from '@inertiajs/svelte'
  import FlashBanner from './FlashBanner.svelte'

  export let site = {}
  export let flash = {}
  export let labels = {}

  const nav = [
    { href: '/dashboard', label: 'Dashboard' },
    { href: '/sales', label: 'Vendas' },
    { href: '/products', label: 'Produtos' },
    { href: '/finance', label: 'Financeiro' },
    { href: '/affiliates', label: 'Afiliados' },
    { href: '/webhooks', label: 'Webhooks' },
    { href: '/events', label: 'Eventos' },
    { href: '/audit', label: 'Auditoria' },
    { href: '/account', label: 'Conta' },
    { href: '/settings', label: 'Configurações' },
  ]

  function isActive(href) {
    const url = page.url || ''
    const path = url.split('?')[0]
    if (href === '/dashboard') {
      return path === '/dashboard' || path === '/'
    }
    return path === href || path.startsWith(href + '/')
  }

  function logout() {
    router.post('/logout')
  }
</script>

<div class="min-h-screen flex bg-[#f0fdf4] text-green-950">
  <!-- Sidebar -->
  <aside
    class="flex w-60 shrink-0 flex-col bg-[#14532d] text-green-50"
    data-testid="app-sidebar"
  >
    <div class="border-b border-green-800/60 px-5 py-5">
      <a
        href="/dashboard"
        use:inertia
        class="block text-lg font-semibold tracking-tight text-white"
      >
        {site.appName || 'Kiwify Ops'}
      </a>
      <p class="mt-0.5 text-xs text-green-300/80">Operações</p>
    </div>

    <nav class="flex-1 space-y-0.5 overflow-y-auto px-3 py-4 text-sm">
      {#each nav as item}
        <a
          href={item.href}
          use:inertia
          class="block rounded-lg px-3 py-2 transition-colors {isActive(item.href)
            ? 'bg-[#166534] font-medium text-white ring-1 ring-[#22c55e]/40'
            : 'text-green-100/90 hover:bg-[#166534]/70 hover:text-white'}"
          aria-current={isActive(item.href) ? 'page' : undefined}
        >
          {item.label}
        </a>
      {/each}
    </nav>

    <div class="border-t border-green-800/60 p-3">
      <button
        type="button"
        class="w-full rounded-lg px-3 py-2 text-left text-sm text-green-100/90 transition-colors hover:bg-[#166534]/70 hover:text-white"
        data-testid="logout-button"
        on:click={logout}
      >
        Sair
      </button>
    </div>
  </aside>

  <!-- Main -->
  <div class="flex min-w-0 flex-1 flex-col">
    <main class="flex-1 p-6 md:p-8">
      <FlashBanner {flash} />
      <slot />
    </main>
  </div>
</div>
