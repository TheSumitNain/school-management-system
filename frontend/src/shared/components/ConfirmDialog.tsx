import { Modal } from "./Modal";

type Props = {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  pending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ open, title, description, confirmLabel = "Confirm", pending = false, onConfirm, onCancel }: Props) {
  return (
    <Modal open={open} title={title} description={description} onClose={onCancel} closeDisabled={pending}>
      <div className="app-modal-actions">
        <button type="button" className="app-modal-cancel" onClick={onCancel} disabled={pending}>Cancel</button>
        <button type="button" className="app-modal-confirm" onClick={onConfirm} disabled={pending}>{pending ? "Please wait…" : confirmLabel}</button>
      </div>
    </Modal>
  );
}
