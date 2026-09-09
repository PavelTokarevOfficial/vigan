import type { ClassValue } from 'clsx'
import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

// Shared by locally generated shadcn-vue components to merge Tailwind classes.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
