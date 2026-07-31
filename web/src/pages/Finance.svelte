<script>
  import { router } from '@inertiajs/svelte'
  import AppLayout from '../components/AppLayout.svelte'
  import ConfirmModal from '../components/ConfirmModal.svelte'
  import DataTable from '../components/DataTable.svelte'
  import StatCard from '../components/StatCard.svelte'

  export let balances = null
  export let payouts = []
  export let pagination = {}
  export let errors = {}
  export let site = {}
  export let flash = {}

  let amountReais = ''
  let confirmOpen = false
  let submitting = false

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
      paid: 'Pago',
      pending: 'Pendente',
      processing: 'Processando',
      failed: 'Falhou',
      cancelled: 'Cancelado',
      canceled: 'Cancelado',
    }
    return map[status] || status || '—'
  }

  /** Parse user input (reais) → centavos for the API. */
  function amountToCentavos(raw) {
    if (raw == null || raw === '') return null
    const normalized = String(raw).trim().replace(',', '.')
    const reais = Number(normalized)
    if (!Number.isFinite(reais) || reais <= 0) return null
    return Math.round(reais * 100)
  }

  $: centavos = amountToCentavos(amountReais)
  $: canSubmit = centavos != null && centavos > 0 && !submitting

  function openConfirm() {
    if (!canSubmit) return
    confirmOpen = true
  }

  function cancelConfirm() {
    confirmOpen = false
  }

  function confirmPayout() {
    if (centavos == null || submitting) return
    submitting = true
    router.post(
      '/finance/payouts',
      { amount: centavos },
      {
        onFinish: () => {
          submitting = false
          confirmOpen = false
          amountReais = ''
        },
      },
    )
  }

  $: errorMessages = Object.values(errors || {}).filter(Boolean)
  $: totalCount = pagination?.count ?? payouts?.length ?? 0
  $: availableLabel = balances != null ? formatBRL(balances.available) : '—'
  $: pendingLabel = balances != null ? formatBRL(balances.pending) : '—'
</script>

<svelte:head>
  <title>Financeiro · {site.appName || 'Kiwify Ops'}</title>
</svelte:head>

<AppLayout {site} {flash}>
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-green-950">Financeiro</h1>
    <p class="mt-1 text-sm text-green-800/70">
      Saldos e saques da conta Kiwify
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

  <div class="mb-8 grid gap-4 sm:grid-cols-2">
    <StatCard
      title="Saldo disponível"
      value={availableLabel}
      hint={errors?.balances || 'Pronto para saque'}
    />
    <StatCard
      title="Saldo pendente"
      value={pendingLabel}
      hint={errors?.balances || 'Em processamento'}
    />
  </div>

  <section class="mb-10 rounded-xl border border-[#bbf7d0] bg-white p-5 shadow-sm">
    <h2 class="text-lg font-semibold text-green-950">Solicitar saque</h2>
    <p class="mt-1 text-sm text-green-800/70">
      Informe o valor em reais. A solicitação é registrada na auditoria.
    </p>
    <form
      class="mt-4 flex flex-wrap items-end gap-3"
      on:submit|preventDefault={openConfirm}
      data-testid="payout-form"
    >
      <div>
        <label for="amount" class="mb-1 block text-xs font-medium text-green-900">
          Valor (R$)
        </label>
        <input
          id="amount"
          type="number"
          min="0.01"
          step="0.01"
          bind:value={amountReais}
          placeholder="0,00"
          class="w-40 rounded-lg border border-green-200 px-3 py-2 text-sm focus:border-green-600 focus:outline-none focus:ring-1 focus:ring-green-600"
          data-testid="payout-amount"
        />
      </div>
      <button
        type="submit"
        disabled={!canSubmit}
        class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d] disabled:cursor-not-allowed disabled:opacity-50"
        data-testid="payout-submit"
      >
        Solicitar saque
      </button>
    </form>
  </section>

  <div class="mb-3 flex items-end justify-between gap-3">
    <h2 class="text-lg font-semibold text-green-950">Histórico de saques</h2>
    <p class="text-xs text-green-700/70">{totalCount} registro(s)</p>
  </div>

  <DataTable headers={['ID', 'Status', 'Valor', 'Criado em']}>
    {#if !payouts || payouts.length === 0}
      <tr>
        <td colspan="4" class="px-4 py-8 text-center text-sm text-green-700/60">
          {#if errors?.payouts || errors?.client}
            {errors.payouts || errors.client}
          {:else}
            Nenhum saque encontrado.
          {/if}
        </td>
      </tr>
    {:else}
      {#each payouts as p}
        <tr class="hover:bg-green-50/40">
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap font-mono text-xs">
            {p.id || '—'}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {statusLabel(p.status)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatBRL(p.amount)}
          </td>
          <td class="px-4 py-3 text-sm text-green-950/90 whitespace-nowrap">
            {formatDate(p.created_at)}
          </td>
        </tr>
      {/each}
    {/if}
  </DataTable>
</AppLayout>

<ConfirmModal
  open={confirmOpen}
  title="Confirmar saque"
  confirmLabel={submitting ? 'Enviando…' : 'Confirmar saque'}
  cancelLabel="Cancelar"
  onconfirm={confirmPayout}
  oncancel={cancelConfirm}
>
  <p>
    Tem certeza de que deseja solicitar o saque de
    <strong>{centavos != null ? formatBRL(centavos) : '—'}</strong>?
    Esta ação é registrada na auditoria.
  </p>
</ConfirmModal>
