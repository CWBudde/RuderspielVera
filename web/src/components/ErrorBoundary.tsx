import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui/button";

/** Catches render errors so a bad view never leaves a blank page; the game itself is safe on the server. */
export class ErrorBoundary extends Component<{ children: ReactNode; onHome?: () => void }, { error: Error | null }> {
  state = { error: null as Error | null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("render error", error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="grid min-h-dvh place-items-center p-6 text-center">
        <div className="max-w-sm rounded-[20px] bg-cream p-5 cut">
          <p className="font-display text-xl font-bold">Hier ist etwas schiefgegangen</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Dein Spielstand ist gespeichert. Lade die Seite neu, um weiterzuspielen.
          </p>
          <p className="mt-2 font-mono text-xs break-words text-muted-foreground">{this.state.error.message}</p>
          <div className="mt-4 flex flex-col gap-2">
            <Button onClick={() => window.location.reload()}>Neu laden</Button>
            {this.props.onHome && (
              <Button
                variant="outline"
                onClick={() => {
                  this.setState({ error: null });
                  this.props.onHome?.();
                }}
              >
                Zum Start
              </Button>
            )}
          </div>
        </div>
      </div>
    );
  }
}
