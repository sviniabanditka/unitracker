// Curated catalog of icons available to trackers. Backend stores the icon
// key as a free string so the catalog can grow without a migration; the
// frontend renders nothing if the key is unknown (graceful fallback).
import type { FunctionalComponent } from 'vue'
import {
  Activity,
  Apple,
  Baby,
  Banana,
  Bed,
  BookOpen,
  Bookmark,
  Calendar,
  Camera,
  Clock,
  Cookie,
  Droplet,
  Droplets,
  Flag,
  Footprints,
  Frown,
  Gamepad2,
  Gift,
  Heart,
  HeartPulse,
  Meh,
  Milk,
  Moon,
  Music,
  Pill,
  Ruler,
  Scale,
  Smile,
  Soup,
  Sparkles,
  Star,
  Stethoscope,
  Sun,
  Syringe,
  Thermometer,
  TrendingUp,
  Utensils,
} from 'lucide-vue-next'

type IconComponent = FunctionalComponent

export const TRACKER_ICONS: Record<string, IconComponent> = {
  // Feeding
  baby: Baby,
  milk: Milk,
  apple: Apple,
  banana: Banana,
  cookie: Cookie,
  soup: Soup,
  utensils: Utensils,
  // Sleep
  moon: Moon,
  bed: Bed,
  sun: Sun,
  // Diaper
  droplet: Droplet,
  droplets: Droplets,
  // Activity
  activity: Activity,
  sparkles: Sparkles,
  music: Music,
  footprints: Footprints,
  gamepad: Gamepad2,
  book: BookOpen,
  // Health
  stethoscope: Stethoscope,
  pill: Pill,
  syringe: Syringe,
  thermometer: Thermometer,
  'heart-pulse': HeartPulse,
  // Growth
  ruler: Ruler,
  scale: Scale,
  'trending-up': TrendingUp,
  // Mood
  smile: Smile,
  frown: Frown,
  meh: Meh,
  heart: Heart,
  star: Star,
  // Misc
  calendar: Calendar,
  clock: Clock,
  bookmark: Bookmark,
  flag: Flag,
  camera: Camera,
  gift: Gift,
}

export type TrackerIconKey = keyof typeof TRACKER_ICONS

export const TRACKER_ICON_KEYS = Object.keys(TRACKER_ICONS) as TrackerIconKey[]

export function isKnownIcon(name: string | null | undefined): name is TrackerIconKey {
  return !!name && name in TRACKER_ICONS
}

export function trackerIconComponent(name: string | null | undefined): IconComponent | null {
  if (!name) return null
  const key = name.toLowerCase().trim()
  return TRACKER_ICONS[key] ?? null
}
