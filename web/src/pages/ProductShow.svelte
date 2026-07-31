<script>
  import { inertia } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'

  export let product = null
  export let errors = {}
  export let site = {}
  export let flash = {}

  /** API amounts are in centavos (minor units). */
  function formatBRL(amount) {
    if (amount == null || amount === '' || Number.isNaN(Number(amount))) return '—'
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: 'BRL',
    }).format(Number(amount) / 100)
  }

  function formatDate(iso) {
    if (!iso) return '—'
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return iso
    return d.toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  function statusLabel(status) {
    const map = {
      active: 'Ativo',
      inactive: 'Inativo',
      draft: 'Rascunho',
      archived: 'Arquivado',
    }
    return map[status] || status || '—'
  }

  function typeLabel(type) {
    const map = {
      membership: 'Área de membros',
      course: 'Curso',
      ebook: 'E-book',
      physical: 'Físico',
      service: 'Serviço',
    }
    return map[type] || type || '—'
  }

  function boolLabel(v) {
    if (v === true) return 'Sim'
    if (v === false) return 'Não'
    return '—'
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)

  const fields = [
    { label: 'ID', get: (p) => p.id },
    { label: 'Nome', get: (p) => p.name || '—' },
    { label: 'Status', get: (p) => statusLabel(p.status) },
    { label: 'Tipo', get: (p) => typeLabel(p.type) },
    { label: 'Preço', get: (p) => formatBRL(p.price) },
    { label: 'Moeda', get: (p) => p.currency || 'BRL' },
    { label: 'Tipo de pagamento', get: (p) => p.payment_type || '—' },
    { label: 'Afiliados habilitados', get: (p) => boolLabel(p.affiliate_enabled) },
    { label: 'Criado em', get: (p) => formatDate(p.created_at) },
  ]
</script>

<svelte:head>
  <title>
    {product?.name || product?.id || 'Produto'} · {site.appName || 'Kiwify Ops'}
  </title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <a
      href="/products"
      use:inertia
      class="text-sm font-medium text-[#166534] hover:underline"
    >
      ← Voltar para produtos
    </a>
    <h1 class="mt-2 text-2xl font-semibold text-green-950">
      {#if product}
        {product.name || product.id}
      {:else}
        Detalhe do produto
      {/if}
    </h1>
    <p class="mt-1 text-xs text-green-700/70">
      Somente leitura — criar ou editar produtos apenas no dashboard da Kiwify.
    </p>
  </div>

  {#if errorMessages.length > 0}
    <div
      class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900"
      role="status"
    >
      <ul class="list-inside list-disc">
        {#each errorMessages as msg}
          <li>{msg}</li>
        {/each}
      </ul>
    </div>
  {/if}

  {#if product}
    <div class="overflow-hidden rounded-xl border border-[#bbf7d0] bg-white shadow-sm">
      <dl class="divide-y divide-green-50">
        {#each fields as field}
          <div class="grid gap-1 px-5 py-3 sm:grid-cols-3 sm:gap-4">
            <dt class="text-sm font-medium text-green-800/80">{field.label}</dt>
            <dd class="text-sm text-green-950 sm:col-span-2">{field.get(product)}</dd>
          </div>
        {/each}
      </dl>
    </div>
  {:else if !errorMessages.length}
    <p class="text-sm text-green-700/70">Produto não encontrado.</p>
  {/if}
</AppLayout>
