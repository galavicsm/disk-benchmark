import { useEffect, useRef } from "react";
import type { app } from "../../../wailsjs/go/models";

interface ConfirmationDialogProps {
  assessment: app.WriteImpactAssessment;
  onCancel: () => void;
  onConfirm: () => void;
  returnFocus: HTMLElement | null;
}

export function ConfirmationDialog({
  assessment,
  onCancel,
  onConfirm,
  returnFocus,
}: ConfirmationDialogProps) {
  const dialog = useRef<HTMLElement>(null);
  const confirmButton = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const previousFocus =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;

    confirmButton.current?.focus();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onCancel();
        return;
      }
      if (event.key !== "Tab" || dialog.current === null) {
        return;
      }

      const focusable = Array.from(
        dialog.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ),
      );
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      window.requestAnimationFrame(() => {
        const focusTarget = returnFocus?.isConnected ? returnFocus : previousFocus;
        if (focusTarget?.isConnected) {
          focusTarget.focus();
        }
      });
    };
  }, [onCancel, returnFocus]);

  return (
    <div className="dialog-backdrop">
      <section
        className="confirmation-dialog"
        ref={dialog}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirmation-title"
        aria-describedby="confirmation-description"
      >
        <div className="dialog-icon" aria-hidden="true">
          !
        </div>
        <div>
          <p className="eyebrow">Confirmation required</p>
          <h2 id="confirmation-title">This run has elevated write impact</h2>
          <p id="confirmation-description">
            Review why this configuration may write more data than a typical benchmark.
          </p>
          <ul className="impact-reasons">
            {assessment.reasons.map((reason) => (
              <li key={reason}>{reason}</li>
            ))}
          </ul>
          <p className="dialog-note">
            Continue only if the selected drive has sufficient free space and the write
            endurance impact is acceptable.
          </p>
          <div className="dialog-actions">
            <button type="button" className="button secondary" onClick={onCancel}>
              Go back
            </button>
            <button
              ref={confirmButton}
              type="button"
              className="button danger"
              onClick={onConfirm}
            >
              Confirm and start
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
