import { useEffect, useCallback } from 'react';

type KeyHandler = (e: KeyboardEvent) => void;

interface KeyBinding {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  handler: KeyHandler;
  description?: string;
}

/**
 * Global keyboard shortcut hook.
 * Listens for keydown events on the document.
 * Ignores events when typing in input/textarea/select elements (unless modifier keys are held).
 */
export function useKeyboard(bindings: KeyBinding[]) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't fire shortcuts when typing in form elements (unless modifiers held)
      const tag = (e.target as HTMLElement).tagName;
      const isTyping = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
      const hasModifier = e.ctrlKey || e.metaKey;

      for (const binding of bindings) {
        const keyMatch = e.key.toLowerCase() === binding.key.toLowerCase();
        const ctrlMatch = binding.ctrl ? e.ctrlKey : !binding.ctrl;
        const metaMatch = binding.meta ? e.metaKey : true;
        const shiftMatch = binding.shift ? e.shiftKey : !binding.shift;

        // Cmd/Ctrl+K should always fire even in inputs
        if (hasModifier && keyMatch) {
          if (ctrlMatch || metaMatch) {
            e.preventDefault();
            binding.handler(e);
            return;
          }
        }

        // Other shortcuts: skip if typing
        if (isTyping) continue;

        if (keyMatch && shiftMatch && !binding.ctrl && !binding.meta) {
          e.preventDefault();
          binding.handler(e);
          return;
        }
      }
    },
    [bindings]
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}
