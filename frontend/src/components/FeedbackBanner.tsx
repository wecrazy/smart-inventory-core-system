type FeedbackBannerProps = {
  tone: 'success' | 'error';
  message: string;
  onClose: () => void;
};

export function FeedbackBanner({ tone, message, onClose }: FeedbackBannerProps) {
  return (
    <div className={`feedback-banner feedback-${tone}`} role={tone === 'error' ? 'alert' : 'status'}>
      <p className="feedback-copy">{message}</p>
      <button aria-label="Dismiss notification" className="feedback-close" onClick={onClose} type="button">
        Close
      </button>
    </div>
  );
}