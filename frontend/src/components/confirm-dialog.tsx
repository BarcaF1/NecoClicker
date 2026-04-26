import * as React from 'react'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

type ConfirmReq = {
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  destructive?: boolean
}

type Ctx = {
  ask: (req: ConfirmReq) => Promise<boolean>
  alert: (title: string, description?: string) => Promise<void>
}

const ConfirmContext = React.createContext<Ctx>({
  ask: async () => false,
  alert: async () => {},
})

export function ConfirmProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = React.useState(false)
  const [req, setReq] = React.useState<ConfirmReq | null>(null)
  const [resolver, setResolver] = React.useState<((v: boolean) => void) | null>(null)
  const [alertMode, setAlertMode] = React.useState(false)

  const ask = React.useCallback((r: ConfirmReq) => {
    setReq(r)
    setOpen(true)
    setAlertMode(false)
    return new Promise<boolean>((resolve) => setResolver(() => resolve))
  }, [])

  const alertFn = React.useCallback((title: string, description?: string) => {
    setReq({ title, description, confirmText: 'OK' })
    setOpen(true)
    setAlertMode(true)
    return new Promise<void>((resolve) => {
      setResolver(() => () => resolve())
    })
  }, [])

  const close = (val: boolean) => {
    if (resolver) resolver(val)
    setOpen(false)
    setResolver(null)
    // small timeout чтобы Radix успел закрыть до сброса контента
    setTimeout(() => setReq(null), 200)
  }

  return (
    <ConfirmContext.Provider value={{ ask, alert: alertFn }}>
      {children}
      <Dialog open={open} onOpenChange={(o) => { if (!o) close(false) }}>
        <DialogContent>
          <DialogHeader>
            <div className="mb-2 flex items-center gap-2">
              {req?.destructive && (
                <span className="flex h-9 w-9 items-center justify-center rounded-full bg-destructive/15 text-destructive">
                  <AlertTriangle className="h-5 w-5" />
                </span>
              )}
              <DialogTitle>{req?.title || ''}</DialogTitle>
            </div>
            {req?.description && (
              <DialogDescription>{req.description}</DialogDescription>
            )}
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-2">
            {!alertMode && (
              <Button variant="outline" onClick={() => close(false)}>
                {req?.cancelText || 'Отмена'}
              </Button>
            )}
            <Button
              variant={req?.destructive ? 'destructive' : 'default'}
              onClick={() => close(true)}
              autoFocus
            >
              {req?.confirmText || 'OK'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ConfirmContext.Provider>
  )
}

export function useConfirm() {
  return React.useContext(ConfirmContext)
}
