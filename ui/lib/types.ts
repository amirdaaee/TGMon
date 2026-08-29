export type MediaFileMeta = {
  FileSize: number;
  FileName: string;
  MimeType: string;
  FileID: number;
  Duration: number;
};

export type MediaFileDoc = {
  ID: string;
  CreatedAt?: string;
  UpdatedAt?: string;
  DeletedAt?: string;
  Meta: MediaFileMeta;
  MessageID: number;
  Thumbnail: string;
  Vtt: string;
  Sprite: string;
  Srt: string;
  HasHash: boolean;
  UName: string;
};

export type MediaExtendedMeta = {
  ID: string;
  CreatedAt?: string;
  UpdatedAt?: string;
  DeletedAt?: string;
  MediaFileID: string;
  LastPlayedAt: string;
  Checkpoint: number;
  Score: number;
  Likes: number;
  IsFavorite: boolean;
};

export type MediaWithMeta = {
  Media: MediaFileDoc;
  Meta: MediaExtendedMeta;
};

export type MediaListRes = {
  Media: MediaWithMeta[] | null;
  Total: number;
};

export type MediaReadRes = MediaWithMeta & {
  pervID: string | null;
  nextID: string | null;
};

export type LoginPostReq = {
  Username: string;
  Password: string;
};

export type LoginPostRes = {
  Token: string;
};

export type MediaMetaPatchReq = {
  LastPlayedAt?: string;
  Checkpoint?: number;
  Score?: number;
  Likes?: number;
  IsFavorite?: boolean;
};

export type ApiErrorBody = {
  statusCode?: number;
  msg?: string;
  message?: string;
};
