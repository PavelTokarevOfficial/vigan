export type LayerType =
  | 'input_video'
  | 'asset_video'
  | 'image'
  | 'gif'
  | 'subtitles'
  | 'text'
  | 'audio'
  | 'color'

export type Layer = {
  id: string
  name: string
  type: LayerType
  x: number
  y: number
  width: number
  height: number
  visible: boolean
  opacity: number
  fit?: 'cover' | 'contain' | 'stretch'
  assetId?: string
  text?: string
  color?: string
  filters?: { blur?: number; brightness?: number }
  style?: {
    fontSize?: number
    alignment?: number
    marginV?: number
    outline?: number
    primaryColor?: string
    outlineColor?: string
  }
}

export type TemplateConfig = {
  version: 1
  canvas: { width: number; height: number; fps: number; background: string }
  layers: Layer[]
}

export type VideoTemplate = {
  id: string
  name: string
  description: string
  previewAssetId: string | null
  previewUrl?: string
  configVersion: number
  config: TemplateConfig
  createdAt: string
  updatedAt: string
}

export function createDefaultConfig(): TemplateConfig {
  return {
    version: 1,
    canvas: { width: 1080, height: 1920, fps: 30, background: '#000000' },
    layers: [
      {
        id: 'background',
        name: 'Размытый фон',
        type: 'input_video',
        x: 0,
        y: 0,
        width: 1080,
        height: 1920,
        visible: true,
        opacity: 1,
        fit: 'cover',
        filters: { blur: 25, brightness: -0.2 },
      },
      {
        id: 'clip',
        name: 'Основной клип',
        type: 'input_video',
        x: 0,
        y: 0,
        width: 1080,
        height: 1920,
        visible: true,
        opacity: 1,
        fit: 'contain',
      },
      {
        id: 'subtitles',
        name: 'Субтитры',
        type: 'subtitles',
        x: 90,
        y: 1540,
        width: 900,
        height: 240,
        visible: true,
        opacity: 1,
        style: {
          fontSize: 8,
          alignment: 2,
          marginV: 100,
          outline: 2,
          primaryColor: '&H00FFFFFF',
          outlineColor: '&H00000000',
        },
      },
    ],
  }
}
