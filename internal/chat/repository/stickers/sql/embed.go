package stickerssql

import _ "embed"

//go:embed get_all_sticker_packs.sql
var GetAllStickerPacks string

//go:embed get_stickers_by_pack_ids.sql
var GetStickersByPackIDs string

//go:embed get_sticker_by_id.sql
var GetStickerByID string

//go:embed get_stickers_by_ids.sql
var GetStickersByIDs string
