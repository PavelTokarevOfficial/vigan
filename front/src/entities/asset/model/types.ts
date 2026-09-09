export type AssetKind = 'image' | 'gif' | 'video' | 'audio'

export type AssetFolder = {
  id: string
  parentId: string | null
  name: string
  createdAt: string
  updatedAt: string
}

export type Asset = {
  id: string
  folderId: string | null
  name: string
  kind: AssetKind
  mimeType: string
  size: number
  storageKey: string
  url?: string
  createdAt: string
  updatedAt: string
}

export type AssetLibrary = { folders: AssetFolder[]; assets: Asset[] }
