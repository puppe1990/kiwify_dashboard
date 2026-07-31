<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import ConfirmModal from '../components/ConfirmModal.svelte'

  export let sale = null
  export let errors = {}
  export let site = {}
  export let flash = {}

  let confirmOpen = false
  let pixKey = ''
  let submitting = false

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
      paid: 'Pago',
      waiting_payment: 'Aguardando',
      refused: 'Recusado',
      refunded: 'Reembolsado',
      chargedback: 'Chargeback',
      pending: 'Pendente',
    }
    return map[status] || status || '—'
  }

  function paymentLabel(method) {
    const map = {
      credit_card: 'Cartão de crédito',
      pix: 'Pix',
      boleto: 'Boleto',
    }
    return map[method] || method || '—'
  }

  function openRefund() {
    confirmOpen = true
  }

  function cancelRefund() {
    confirmOpen = false
  }

  function confirmRefund() {
    if (!sale?.id || submitting) return
    submitting = true
    router.post(
      `/sales/${sale.id}/refund`,
      { pixKey: pixKey || '' },
      {
        onFinish: () => {
          submitting = false
          confirmOpen = false
          pixKey = ''
        },
      },
    )
  }

  $: canRefund =
    sale &&
    sale.status &&
    !['refunded', 'chargedback', 'refused'].includes(String(sale.status).toLowerCase())

  $: errorMessages = Object.values(errors || {}).filter(Boolean)

  const fields = [
    { label: 'ID', get: (s) => s.id },
    { label: 'Referência', get: (s) => s.reference },
    { label: 'Status', get: (s) => statusLabel(s.status) },
    { label: 'Tipo', get: (s) => s.type || '—' },
    { label: 'Pagamento', get: (s) => paymentLabel(s.payment_method) },
    { label: 'Valor líquido', get: (s) => formatBRL(s.net_amount) },
    { label: 'Moeda', get: (s) => s.currency || 'BRL' },
    { label: 'Criada em', get: (s) => formatDate(s.created_at) },
    { label: 'Atualizada em', get: (s) => formatDate(s.updated_at) },
    { label: 'Produto', get: (s) => s.product?.name || s.product?.id || '—' },
    { label: 'Cliente', get: (s) => s.customer?.name || '—' },
    { label: 'E-mail', get: (s) => s.customer?.email || '—' },
  ]
</script>

<svelte:head>
  <title>
    {sale?.reference || sale?.id || 'Venda'} · {site.appName || 'Kiwify Ops'}
  </title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <a
      href="/sales"
      use:inertia
      class="text-sm font-medium text-[#166534] hover:underline"
    >
      ← Voltar para vendas
    </a>
    <h1 class="mt-2 text-2xl font-semibold text-green-950">
      {#if sale}
        Venda {sale.reference || sale.id}
      {:else}
        Detalhe da venda
      {/if}
    </h1>
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

  {#if sale}
    <div class="overflow-hidden rounded-xl border border-[#bbf7d0] bg-white shadow-sm">
      <dl class="divide-y divide-green-50">
        {#each fields as field}
          <div class="grid gap-1 px-5 py-3 sm:grid-cols-3 sm:gap-4">
            <dt class="text-sm font-medium text-green-800/80">{field.label}</dt>
            <dd class="text-sm text-green-950 sm:col-span-2">{field.get(sale)}</dd>
          </div>
        {/each}
      </dl>
    </div>

    <div class="mt-6 flex flex-wrap gap-3">
      {#if canRefund}
        <button
          type="button"
          class="rounded-lg bg-red-700 px-4 py-2 text-sm font-medium text-white hover:bg-red-800"
          data-testid="refund-button"
          on:click={openRefund}
        >
          Reembolsar venda
        </button>
      {:else}
        <p class="text-sm text-green-700/70">
          Esta venda não está elegível para reembolso por este painel.
        </p>
      {/if}
    </div>
  {:else if !errorMessages.length}
    <p class="text-sm text-green-700/70">Venda não encontrada.</p>
  {/if}
</AppLayout>

<ConfirmModal
  open={confirmOpen}
  title="Confirmar reembolso"
  confirmLabel={submitting ? 'Enviando…' : 'Confirmar reembolso'}
  cancelLabel="Cancelar"
  onconfirm={confirmRefund}
  oncancel={cancelRefund}
>
  <p class="mb-3">
    Tem certeza de que deseja reembolsar a venda
    <strong>{sale?.reference || sale?.id}</strong>?
    Esta ação é registrada na auditoria.
  </p>
  <label for="pix_key" class="mb-1 block text-xs font-medium text-green-900">
    Chave Pix (opcional)
  </label>
  <input
    id="pix_key"
    type="text"
    bind:value={pixKey}
    placeholder="Somente se a API exigir para este pagamento"
    autocomplete="off"
    class="block w-full rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
  />
</ConfirmModal>
