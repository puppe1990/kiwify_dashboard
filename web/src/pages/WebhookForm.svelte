<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import ConfirmModal from '../components/ConfirmModal.svelte'

  export let webhook = null
  export let triggers = []
  export let defaultUrl = ''
  export let form = null
  export let errors = {}
  export let site = {}
  export let flash = {}

  const triggerLabels = {
    boleto_gerado: 'Boleto gerado',
    pix_gerado: 'PIX gerado',
    carrinho_abandonado: 'Carrinho abandonado',
    compra_recusada: 'Compra recusada',
    compra_aprovada: 'Compra aprovada',
    compra_reembolsada: 'Compra reembolsada',
    chargeback: 'Chargeback',
    subscription_canceled: 'Assinatura cancelada',
    subscription_late: 'Assinatura atrasada',
    subscription_renewed: 'Assinatura renovada',
  }

  let name = ''
  let url = ''
  let products = 'all'
  let productId = ''
  let token = ''
  /** @type {Record<string, boolean>} */
  let selected = {}
  let submitting = false
  let confirmDeleteOpen = false
  let confirmSaveOpen = false

  $: isEdit = !!(webhook && webhook.id)

  $: {
    const src = form || webhook || {}
    name = src.name || ''
    url = src.url || (!isEdit ? defaultUrl || '' : '')
    const prod = src.products || 'all'
    if (prod === 'all' || !prod) {
      products = 'all'
      productId = ''
    } else {
      products = 'id'
      productId = prod
    }
    token = form?.token || ''
    const trigs = src.triggers || form?.triggers || []
    const next = {}
    for (const t of triggers || []) {
      next[t] = Array.isArray(trigs) ? trigs.includes(t) : false
    }
    selected = next
  }

  function productsValue() {
    if (products === 'all') return 'all'
    return (productId || '').trim() || 'all'
  }

  function selectedTriggers() {
    return Object.entries(selected)
      .filter(([, on]) => on)
      .map(([k]) => k)
  }

  function openSaveConfirm() {
    if (submitting) return
    if (!name.trim() || !url.trim() || selectedTriggers().length === 0) {
      return
    }
    confirmSaveOpen = true
  }

  function cancelSave() {
    confirmSaveOpen = false
  }

  function submitSave() {
    if (submitting) return
    submitting = true
    const payload = {
      name: name.trim(),
      url: url.trim(),
      products: productsValue(),
      triggers: selectedTriggers(),
      token: token.trim() || undefined,
    }
    const opts = {
      onFinish: () => {
        submitting = false
        confirmSaveOpen = false
      },
    }
    if (isEdit) {
      router.post(`/webhooks/${webhook.id}`, payload, opts)
    } else {
      router.post('/webhooks', payload, opts)
    }
  }

  function openDeleteConfirm() {
    if (!isEdit || submitting) return
    confirmDeleteOpen = true
  }

  function cancelDelete() {
    confirmDeleteOpen = false
  }

  function confirmDelete() {
    if (!isEdit || submitting) return
    submitting = true
    router.delete(`/webhooks/${webhook.id}`, {
      onFinish: () => {
        submitting = false
        confirmDeleteOpen = false
      },
    })
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: pageTitle = isEdit ? webhook?.name || 'Editar webhook' : 'Novo webhook'
</script>

<svelte:head>
  <title>{pageTitle} · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <a href="/webhooks" use:inertia class="text-sm font-medium text-[#166534] hover:underline">
      ← Voltar para webhooks
    </a>
    <h1 class="mt-2 text-2xl font-semibold text-green-950">{pageTitle}</h1>
    <p class="mt-1 text-sm text-green-800/70">
      {#if isEdit}
        Edite o webhook na API Kiwify. Alterações são registradas na auditoria.
      {:else}
        Cadastre um webhook apontando para a URL pública de recebimento deste app.
      {/if}
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

  {#if defaultUrl && !isEdit}
    <div
      class="mb-4 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-900"
      data-testid="default-receive-url"
    >
      <p class="font-medium">URL de recebimento deste app</p>
      <p class="mt-1 break-all font-mono text-xs">{defaultUrl}</p>
      <p class="mt-1 text-xs text-green-800/70">
        Prefira usar esta URL para gravar eventos localmente em Eventos.
      </p>
    </div>
  {/if}

  <form
    class="space-y-6 rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm"
    on:submit|preventDefault={openSaveConfirm}
    data-testid="webhook-form"
  >
    <div class="grid gap-4 sm:grid-cols-2">
      <div class="sm:col-span-2">
        <label for="name" class="mb-1 block text-xs font-medium text-green-900">Nome</label>
        <input
          id="name"
          type="text"
          bind:value={name}
          required
          class="block w-full rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
      </div>

      <div class="sm:col-span-2">
        <label for="url" class="mb-1 block text-xs font-medium text-green-900">URL de destino</label>
        <input
          id="url"
          type="url"
          bind:value={url}
          required
          placeholder="https://..."
          class="block w-full rounded-lg border border-green-200 px-3 py-2 font-mono text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
      </div>

      <div>
        <label for="products" class="mb-1 block text-xs font-medium text-green-900">Produtos</label>
        <select
          id="products"
          bind:value={products}
          class="block w-full rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        >
          <option value="all">Todos os produtos</option>
          <option value="id">Produto específico</option>
        </select>
      </div>

      {#if products === 'id'}
        <div>
          <label for="product_id" class="mb-1 block text-xs font-medium text-green-900">
            ID do produto
          </label>
          <input
            id="product_id"
            type="text"
            bind:value={productId}
            class="block w-full rounded-lg border border-green-200 px-3 py-2 font-mono text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
          />
        </div>
      {/if}

      <div class="sm:col-span-2">
        <label for="token" class="mb-1 block text-xs font-medium text-green-900">
          Token (opcional)
        </label>
        <input
          id="token"
          type="text"
          bind:value={token}
          placeholder={isEdit ? 'Deixe em branco para manter' : 'Opcional'}
          class="block w-full rounded-lg border border-green-200 px-3 py-2 font-mono text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
        />
        {#if isEdit && webhook?.token}
          <p class="mt-1 text-xs text-green-700/70">
            Token atual: <span class="font-mono">{webhook.token}</span>
          </p>
        {/if}
      </div>
    </div>

    <fieldset>
      <legend class="mb-2 text-xs font-medium text-green-900">Gatilhos</legend>
      <div class="grid gap-2 sm:grid-cols-2" data-testid="webhook-triggers">
        {#each triggers as t}
          <label class="flex items-center gap-2 rounded-lg border border-green-100 px-3 py-2 text-sm text-green-950 hover:bg-green-50/50">
            <input type="checkbox" bind:checked={selected[t]} class="rounded border-green-300 text-[#166534] focus:ring-green-600" />
            <span>{triggerLabels[t] || t}</span>
            <span class="ml-auto font-mono text-[10px] text-green-700/60">{t}</span>
          </label>
        {/each}
      </div>
    </fieldset>

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-green-100 pt-4">
      <div>
        {#if isEdit}
          <button
            type="button"
            class="rounded-lg border border-red-200 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
            data-testid="webhook-delete"
            on:click={openDeleteConfirm}
            disabled={submitting}
          >
            Excluir
          </button>
        {/if}
      </div>
      <button
        type="submit"
        disabled={submitting}
        class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-50"
        data-testid="webhook-save"
      >
        {isEdit ? 'Salvar alterações' : 'Criar webhook'}
      </button>
    </div>
  </form>
</AppLayout>

<ConfirmModal
  open={confirmSaveOpen}
  title={isEdit ? 'Confirmar atualização' : 'Confirmar criação'}
  confirmLabel={isEdit ? 'Salvar' : 'Criar'}
  cancelLabel="Cancelar"
  onconfirm={submitSave}
  oncancel={cancelSave}
>
  <p>
    {#if isEdit}
      Atualizar o webhook <strong>{name}</strong> apontando para
      <span class="break-all font-mono text-xs">{url}</span>?
    {:else}
      Criar webhook <strong>{name}</strong> com destino
      <span class="break-all font-mono text-xs">{url}</span>?
    {/if}
  </p>
  <p class="mt-2 text-xs text-green-800/70">
    Gatilhos: {selectedTriggers().join(', ') || 'nenhum'} · Produtos: {productsValue()}
  </p>
</ConfirmModal>

<ConfirmModal
  open={confirmDeleteOpen}
  title="Excluir webhook"
  confirmLabel="Excluir"
  cancelLabel="Cancelar"
  onconfirm={confirmDelete}
  oncancel={cancelDelete}
>
  <p>
    Tem certeza que deseja excluir o webhook
    <strong>{webhook?.name || webhook?.id}</strong>?
  </p>
  <p class="mt-2 break-all font-mono text-xs text-green-800/70">{webhook?.url}</p>
  <p class="mt-2 text-xs text-red-700/80">Esta ação não pode ser desfeita na Kiwify.</p>
</ConfirmModal>
