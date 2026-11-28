import { useState } from "react";
import { Check, MonitorSmartphone } from "lucide-react";
import { resumeAvailable, resumeLink, type Session } from "@/api/client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/i18n";

/**
 * "Continue on another device": shares (or copies) the personal resume link of this seat, with a
 * warning that the link hands over the seat. Renders nothing for mock and wasm sessions.
 */
export function ResumeLink({ session, className }: { session: Session; className?: string }) {
  const [copied, setCopied] = useState(false);
  if (!resumeAvailable(session)) return null;

  const share = async () => {
    const url = resumeLink(session);
    try {
      if (navigator.share) {
        await navigator.share({ title: t("app.title"), text: t("resume.shareText"), url });
        return;
      }
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* share sheet dismissed or clipboard blocked */
    }
  };

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <Button type="button" variant="outline" onClick={() => void share()}>
        {copied ? <Check /> : <MonitorSmartphone />}
        {copied ? t("resume.copied") : t("resume.button")}
      </Button>
      <p className="text-xs text-muted-foreground">{t("resume.hint")}</p>
    </div>
  );
}
