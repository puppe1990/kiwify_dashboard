<script>
  /**
   * Confirm dialog — Svelte 5 friendly via callback props.
   * Use default slot for body content.
   */
  export let open = false
  export let title = 'Confirmar'
  export let confirmLabel = 'Confirmar'
  export let cancelLabel = 'Cancelar'
  /** @type {(() => void) | undefined} */
  export let onconfirm = undefined
  /** @type {(() => void) | undefined} */
  export let oncancel = undefined

  function handleConfirm() {
    onconfirm?.()
  }

  function handleCancel() {
    oncancel?.()
  }

  function handleKeydown(e) {
    if (!open) return
    if (e.key === 'Escape') handleCancel()
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="confirm-modal-title"
    data-testid="confirm-modal"
  >
    <button
      type="button"
      class="absolute inset-0 bg-green-950/40 backdrop-blur-[1px]"
      aria-label={cancelLabel}
      on:click={handleCancel}
    ></button>

    <div
      class="relative z-10 w-full max-w-md rounded-xl border border-[#bbf7d0] bg-white p-6 shadow-lg"
    >
      <h2 id="confirm-modal-title" class="text-lg font-semibold text-green-950">
        {title}
      </h2>

      <div class="mt-3 text-sm text-green-900/80">
        <slot />
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <button
          type="button"
          class="rounded-lg border border-green-200 px-4 py-2 text-sm font-medium text-green-900 hover:bg-green-50"
          data-testid="confirm-modal-cancel"
          on:click={handleCancel}
        >
          {cancelLabel}
        </button>
        <button
          type="button"
          class="rounded-lg bg-[#166534] px-4 py-2 text-sm font-medium text-white hover:bg-[#14532d]"
          data-testid="confirm-modal-confirm"
          on:click={handleConfirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
