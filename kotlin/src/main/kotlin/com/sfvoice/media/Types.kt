package com.sfvoice.media

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.KSerializer
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive

// ── enums ─────────────────────────────────────────────────────────────────

@Serializable
enum class MediaType {
    @SerialName("video") Video,
    @SerialName("audio") Audio,
}

@Serializable
enum class SourceType {
    @SerialName("url") Url,
    @SerialName("s3") S3,
    @SerialName("file") File,
    @SerialName("upload") Upload,
    @SerialName("youtube") Youtube,
    @SerialName("twitter") Twitter,
}

typealias AssetSourceType = SourceType

@Serializable
enum class TaskStatus {
    @SerialName("pending")  Pending,
    @SerialName("indexing") Indexing,
    @SerialName("ready")    Ready,
    @SerialName("failed")   Failed;

    val isTerminal: Boolean get() = this == Ready || this == Failed
}

@Serializable
enum class SearchMatchType {
    @SerialName("visual")         Visual,
    @SerialName("conversation")   Conversation,
    @SerialName("text_in_video")  TextInVideo,
    @SerialName("video")          Video,
    @SerialName("audio")          Audio,
    @SerialName("transcript")     Transcript,
}

// ── pagination ────────────────────────────────────────────────────────────

@Serializable
data class PageInfo(
    val total: Long,
    val page: Int,
    val limit: Int,
    @SerialName("next_page_token") val nextPageToken: String? = null,
)

// ── asset ─────────────────────────────────────────────────────────────────

@Serializable(with = AssetSerializer::class)
data class Asset(
    @SerialName("asset_id") val id: String,
    @SerialName("media_type")  val mediaType: MediaType,
    @SerialName("source_type") val sourceType: SourceType,
    val status: TaskStatus,
    val metadata: Map<String, String>? = null,
    @SerialName("duration_ms") val durationMs: Long? = null,
    @SerialName("created_at")  val createdAt: String,
    @SerialName("updated_at")  val updatedAt: String,
    val filename: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    val types: List<SearchMatchType> = emptyList(),
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    @SerialName("thumbnail_status") val thumbnailStatus: String? = null,
    val metadataValues: JsonObject? = metadata?.let(::stringMetadataToJson),
) {
    val assetId: String get() = id
}

@Serializable
private data class AssetWire(
    @SerialName("asset_id") val id: String,
    @SerialName("media_type") val mediaType: MediaType,
    @SerialName("source_type") val sourceType: SourceType,
    val status: TaskStatus,
    val metadata: JsonObject? = null,
    @SerialName("duration_ms") val durationMs: Long? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val filename: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    val types: List<SearchMatchType> = emptyList(),
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
    @SerialName("thumbnail_status") val thumbnailStatus: String? = null,
)

object AssetSerializer : KSerializer<Asset> {
    override val descriptor = AssetWire.serializer().descriptor

    override fun deserialize(decoder: Decoder): Asset {
        val wire = decoder.decodeSerializableValue(AssetWire.serializer())
        val stringMetadata = wire.metadata?.mapNotNull { (key, value) ->
            (value as? JsonPrimitive)
                ?.takeIf { it.isString }
                ?.content
                ?.let { key to it }
        }?.toMap()
        return Asset(
            id = wire.id,
            mediaType = wire.mediaType,
            sourceType = wire.sourceType,
            status = wire.status,
            metadata = stringMetadata,
            durationMs = wire.durationMs,
            createdAt = wire.createdAt,
            updatedAt = wire.updatedAt,
            filename = wire.filename,
            projectId = wire.projectId,
            assetClass = wire.assetClass,
            types = wire.types,
            thumbnailUrl = wire.thumbnailUrl,
            thumbnailStatus = wire.thumbnailStatus,
            metadataValues = wire.metadata,
        )
    }

    override fun serialize(encoder: Encoder, value: Asset) {
        val rawStringMetadata = value.metadataValues?.mapNotNull { (key, item) ->
            (item as? JsonPrimitive)
                ?.takeIf { it.isString }
                ?.content
                ?.let { key to it }
        }?.toMap()
        val metadataValues = if (rawStringMetadata == value.metadata) {
            value.metadataValues
        } else {
            value.metadata?.let(::stringMetadataToJson)
        }
        encoder.encodeSerializableValue(
            AssetWire.serializer(),
            AssetWire(
                id = value.id,
                mediaType = value.mediaType,
                sourceType = value.sourceType,
                status = value.status,
                metadata = metadataValues,
                durationMs = value.durationMs,
                createdAt = value.createdAt,
                updatedAt = value.updatedAt,
                filename = value.filename,
                projectId = value.projectId,
                assetClass = value.assetClass,
                types = value.types,
                thumbnailUrl = value.thumbnailUrl,
                thumbnailStatus = value.thumbnailStatus,
            ),
        )
    }
}

private fun stringMetadataToJson(metadata: Map<String, String>): JsonObject =
    JsonObject(metadata.mapValues { (_, value) -> JsonPrimitive(value) })

@Serializable
data class AssetListResponse(
    val items: List<Asset>,
    @SerialName("page_info") val pageInfo: PageInfo,
)

// ── ingest ────────────────────────────────────────────────────────────────

@Serializable
data class IngestRequest(
    val source: String,
    val url: String? = null,
    @SerialName("s3_key")    val s3Key: String? = null,
    @SerialName("media_kind") val mediaType: MediaType? = null,
    val metadata: Map<String, String>? = null,
    @SerialName("asset_id") val assetId: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    val prefix: String? = null,
    @SerialName("transcript_only") val transcriptOnly: Boolean? = null,
) {
    val mediaKind: MediaType? get() = mediaType

    internal fun normalized(): IngestRequest = copy(
        assetId = assetId ?: url?.let(::compatibilityAssetId) ?: s3Key?.let(::compatibilityAssetId)
        ?: throw IllegalArgumentException("assetId is required when neither url nor s3Key is set"),
    )
}

private fun compatibilityAssetId(value: String): String {
    var hash = 0xcbf29ce484222325UL
    value.encodeToByteArray().forEach { valueByte ->
        hash = (hash xor valueByte.toUByte().toULong()) * 0x100000001b3UL
    }
    return "legacy-${hash.toString(16).padStart(16, '0')}"
}

@Serializable
data class IngestResponse(
    @SerialName("asset_id") val assetId: String,
    @SerialName("task_id")  val taskId: String,
    val status: String,
)

// ── tasks ─────────────────────────────────────────────────────────────────

@Serializable
data class Task(
    @SerialName("task_id")      val taskId: String,
    @SerialName("asset_id")     val assetId: String,
    @SerialName("asset_class")  val assetClass: String? = null,
    val types: List<SearchMatchType>,
    val status: TaskStatus,
    val error: String? = null,
    @SerialName("created_at")   val createdAt: String,
    @SerialName("completed_at") val completedAt: String? = null,
)

// ── search ────────────────────────────────────────────────────────────────

@Serializable
data class SearchRequest(
    val query: String,
    val types: List<SearchMatchType>? = null,
    @SerialName("asset_ids") val assetIds: List<String>? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    val prefix: String? = null,
    val threshold: Float? = null,
    val page: Int? = null,
    val limit: Int? = null,
)

@Serializable
data class SearchResult(
    @SerialName("document_id")   val documentId: String,
    @SerialName("asset_id")      val assetId: String? = null,
    @SerialName("asset_class")   val assetClass: String? = null,
    @SerialName("twelvelabs_media_id") val twelveLabsMediaId: String? = null,
    val score: Float,
    @SerialName("start_ms")      val startMs: Long,
    @SerialName("end_ms")        val endMs: Long,
    @SerialName("match_type")    val matchType: SearchMatchType,
    val text: String? = null,
    @SerialName("thumbnail_url") val thumbnailUrl: String? = null,
)

@Serializable
data class SearchResponse(
    @SerialName("search_query_id") val searchQueryId: String,
    val results: List<SearchResult>,
    @SerialName("page_info") val pageInfo: PageInfo,
)

@Serializable
data class Prefix(
    val prefix: String,
    @SerialName("owned_by_caller") val ownedByCaller: Boolean,
)

@Serializable
data class PrefixListResponse(val items: List<Prefix>)

// ── monitors ─────────────────────────────────────────────────────────────

@Serializable
data class Monitor(
    val id: String,
    val slug: String,
    val text: String,
    val threshold: Float,
    val enabled: Boolean,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
)

@Serializable
data class MonitorListResponse(
    val items: List<Monitor>,
    val total: Long,
)

@Serializable
data class CreateMonitorRequest(
    val text: String,
    val slug: String? = null,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    val threshold: Float? = null,
)

@Serializable
data class UpdateMonitorRequest(
    val text: String? = null,
    @SerialName("asset_class") val assetClass: String? = null,
    val threshold: Float? = null,
    val enabled: Boolean? = null,
)

@Serializable
data class MonitorEvent(
    val id: String,
    @SerialName("monitor_id") val monitorId: String,
    @SerialName("document_id") val documentId: String,
    val matched: Boolean,
    @SerialName("webhook_sent") val webhookSent: Boolean,
    @SerialName("created_at") val createdAt: String,
    @SerialName("asset_id") val assetId: String? = null,
    val score: Float? = null,
    @SerialName("match_detail") val matchDetail: JsonObject? = null,
)

@Serializable
data class MonitorEventListResponse(
    val items: List<MonitorEvent>,
    val total: Long,
)

// ── error envelope ────────────────────────────────────────────────────────

@Serializable
internal data class ApiErrorEnvelope(val error: ApiErrorBody)

@Serializable
internal data class ApiErrorBody(val code: String, val message: String)
