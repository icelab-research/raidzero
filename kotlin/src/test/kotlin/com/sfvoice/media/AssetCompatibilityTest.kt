package com.sfvoice.media

// compile-time coverage for the public 0.1.1 constructor and copy surface.
internal fun legacyAssetConstruction(): Asset {
    val asset = Asset(
        id = "asset-1",
        mediaType = MediaType.Audio,
        sourceType = SourceType.Url,
        status = TaskStatus.Ready,
        metadata = mapOf("title" to "call"),
        createdAt = "2026-11-05T00:00:00Z",
        updatedAt = "2026-11-05T00:00:00Z",
    )
    return asset.copy(metadata = mapOf("title" to "updated call"))
}
