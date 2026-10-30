import { Download } from 'lucide-react';

// The URL is a presigned R2 link that answers with Content-Disposition: attachment.
// That header (not the `download` attribute, which browsers ignore cross-origin)
// makes iOS Safari and Android Chrome save the file instead of opening a player.
export function DownloadLink({
  href,
  filename,
  label = false,
}: {
  href: string;
  filename: string;
  label?: boolean;
}) {
  return (
    <a
      href={href}
      download={filename}
      aria-label={`ดาวน์โหลด ${filename}`}
      className={`flex min-h-11 min-w-11 shrink-0 items-center justify-center gap-1.5 rounded-lg text-xs text-muted transition-colors hover:bg-elevated hover:text-ink ${label ? 'px-3' : ''}`}
    >
      <Download size={16} aria-hidden />
      {label && 'ดาวน์โหลด'}
    </a>
  );
}
