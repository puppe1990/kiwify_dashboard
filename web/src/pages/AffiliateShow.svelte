<script>
  import { inertia, router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import ConfirmModal from '../components/ConfirmModal.svelte'

  export let affiliate = null
  export let errors = {}
  export let site = {}
  export let flash = {}

  let status = ''
  let commissionReais = ''
  let confirmOpen = false
  let submitting = false

  $: if (affiliate) {
    status = affiliate.status || ''
    // Pre-fill commission in reais for UX (API stores centavos).
    if (affiliate.commission != null && affiliate.commission !== '') {
      commissionReais = (Number(affiliate.commission) / 100).toFixed(2)
    }
  }

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

  function statusLabel(s) {
    const map = {
      active: 'Ativo',
      blocked: 'Bloqueado',
      refused: 'Recusado',
      pending: 'Pendente',
    }
    return map[s] || s || '—'
  }

  function openConfirm() {
    if (!affiliate?.affiliate_id || submitting) return
    confirmOpen = true
  }

  function cancelConfirm() {
    confirmOpen = false
  }

  function commissionToCentavos(raw) {
    if (raw == null || raw === '') return null
    const reais = Number(String(raw).trim().replace(',', '.'))
    if (!Number.isFinite(reais) || reais < 0) return null
    return Math.round(reais * 100)
  }

  function confirmEdit() {
    if (!affiliate?.affiliate_id || submitting) return
    submitting = true
    const commission = commissionToCentavos(commissionReais)
    router.post(
      `/affiliates/${affiliate.affiliate_id}`,
      {
        status: status || '',
        // API expects commission in centavos (same unit as GET).
        commission: commission != null ? commission : '',
      },
      {
        onFinish: () => {
          submitting = false
          confirmOpen = false
        },
      },
    )
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)

  const fields = [
    { label: 'ID', get: (a) => a.affiliate_id },
    { label: 'Nome', get: (a) => a.name || '—' },
    { label: 'E-mail', get: (a) => a.email || '—' },
    { label: 'Empresa', get: (a) => a.company_name || '—' },
    { label: 'CPF do responsável', get: (a) => a.director_cpf || '—' },
    { label: 'CNPJ', get: (a) => a.company_cnpj || '—' },
    { label: 'Status', get: (a) => statusLabel(a.status) },
    { label: 'Comissão', get: (a) => formatBRL(a.commission) },
    { label: 'Produto', get: (a) => a.product?.name || a.product?.id || '—' },
    { label: 'Criado em', get: (a) => formatDate(a.created_at) },
  ]
</script>

<svelte:head>
  <title>
    {affiliate?.name || affiliate?.affiliate_id || 'Afiliado'} · {site.appName || 'Kiwify Ops'}
  </title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <a
      href="/affiliates"
      use:inertia
      class="text-sm font-medium text-[#166534] hover:underline"
    >
      ← Voltar para afiliados
    </a>
    <h1 class="mt-2 text-2xl font-semibold text-green-950">
      {#if affiliate}
        {affiliate.name || affiliate.affiliate_id}
      {:else}
        Detalhe do afiliado
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

  {#if affiliate}
    <div class="mb-8 overflow-hidden rounded-xl border border-[#bbf7d0] bg-white shadow-sm">
      <dl class="divide-y divide-green-50">
        {#each fields as field}
          <div class="grid gap-1 px-5 py-3 sm:grid-cols-3 sm:gap-4">
            <dt class="text-sm font-medium text-green-800/80">{field.label}</dt>
            <dd class="text-sm text-green-950 sm:col-span-2">{field.get(affiliate)}</dd>
          </div>
        {/each}
      </dl>
    </div>

    <section
      class="rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm"
      data-testid="affiliate-edit-form"
    >
      <h2 class="text-lg font-semibold text-green-950">Editar afiliado</h2>
      <p class="mt-1 text-sm text-green-800/70">
        Campos editáveis pela API: status e comissão. Alterações são registradas na auditoria.
      </p>

      <form class="mt-4 space-y-4" on:submit|preventDefault={openConfirm}>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label for="status" class="mb-1 block text-xs font-medium text-green-900">
              Status
            </label>
            <select
              id="status"
              bind:value={status}
              class="block w-full rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
            >
              <option value="active">Ativo</option>
              <option value="blocked">Bloqueado</option>
              <option value="refused">Recusado</option>
            </select>
          </div>
          <div>
            <label for="commission" class="mb-1 block text-xs font-medium text-green-900">
              Comissão (R$)
            </label>
            <input
              id="commission"
              type="number"
              min="0"
              step="0.01"
              bind:value={commissionReais}
              class="block w-full rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
            />
          </div>
        </div>
        <button
          type="submit"
          disabled={submitting}
          class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d] disabled:opacity-50"
          data-testid="affiliate-save"
        >
          Salvar alterações
        </button>
      </form>
    </section>
  {:else if !errorMessages.length}
    <p class="text-sm text-green-700/70">Afiliado não encontrado.</p>
  {/if}
</AppLayout>

<ConfirmModal
  open={confirmOpen}
  title="Confirmar edição"
  confirmLabel={submitting ? 'Enviando…' : 'Confirmar'}
  cancelLabel="Cancelar"
  onconfirm={confirmEdit}
  oncancel={cancelConfirm}
>
  <p class="mb-2">
    Confirma a atualização do afiliado
    <strong>{affiliate?.name || affiliate?.affiliate_id}</strong>?
  </p>
  <ul class="list-inside list-disc text-sm">
    <li>Status: <strong>{statusLabel(status)}</strong></li>
    <li>
      Comissão:
      <strong>
        {commissionReais !== ''
          ? formatBRL(Math.round(Number(String(commissionReais).replace(',', '.')) * 100))
          : '—'}
      </strong>
    </li>
  </ul>
  <p class="mt-2 text-xs text-green-700/70">Esta ação é registrada na auditoria.</p>
</ConfirmModal>
