import { useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import './AnimatedView.css';

interface AnimatedViewProps {
  children: ReactNode;
}

export default function AnimatedView({ children }: AnimatedViewProps) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    // Use requestAnimationFrame so the browser paints the initial opacity:0 first
    const raf = requestAnimationFrame(() => {
      setVisible(true);
    });
    return () => cancelAnimationFrame(raf);
  }, []);

  return (
    <div className={`animated-view${visible ? ' enter' : ''}`}>
      {children}
    </div>
  );
}
