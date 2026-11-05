package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class IngestRequest {

    private final String source;
    @JsonProperty("asset_id")
    private final String assetId;
    @JsonProperty("asset_class")
    private final String assetClass;
    @JsonProperty("project_id")
    private final String projectId;
    private final String url;
    @JsonProperty("s3_key")
    private final String s3Key;
    @JsonProperty("media_kind")
    private final String mediaKind;
    private final Map<String, String> metadata;
    private final String prefix;
    @JsonProperty("transcript_only")
    private final Boolean transcriptOnly;

    private IngestRequest(Builder builder) {
        this.source    = builder.source;
        this.assetId   = builder.assetId;
        this.assetClass = builder.assetClass;
        this.projectId = builder.projectId;
        this.url       = builder.url;
        this.s3Key     = builder.s3Key;
        this.mediaKind = builder.mediaKind;
        this.metadata  = builder.metadata;
        this.prefix = builder.prefix;
        this.transcriptOnly = builder.transcriptOnly;
    }

    public String getSource()                { return source; }
    public String getAssetId()               { return assetId; }
    public String getAssetClass()            { return assetClass; }
    public String getProjectId()             { return projectId; }
    public String getUrl()                   { return url; }
    public String getS3Key()                 { return s3Key; }
    public String getMediaKind()             { return mediaKind; }
    /** @deprecated use {@link #getMediaKind()} */
    @Deprecated
    public String getMediaType()              { return mediaKind; }
    public Map<String, String> getMetadata() { return metadata; }
    public String getPrefix()                 { return prefix; }
    public Boolean getTranscriptOnly()        { return transcriptOnly; }

    public static Builder fromUrl(String url) {
        return fromUrl(compatibilityAssetId(url), url);
    }

    public static Builder fromUrl(String assetId, String url) {
        return new Builder("url", assetId).url(url);
    }

    public static Builder fromS3(String s3Key) {
        return fromS3(compatibilityAssetId(s3Key), s3Key);
    }

    public static Builder fromS3(String assetId, String s3Key) {
        return new Builder("s3", assetId).s3Key(s3Key);
    }

    private static String compatibilityAssetId(String value) {
        long hash = 0xcbf29ce484222325L;
        for (byte valueByte : value.getBytes(java.nio.charset.StandardCharsets.UTF_8)) {
            hash ^= Byte.toUnsignedLong(valueByte);
            hash *= 0x100000001b3L;
        }
        return String.format("legacy-%016x", hash);
    }

    public static class Builder {
        private final String source;
        private final String assetId;
        private String assetClass;
        private String projectId;
        private String url;
        private String s3Key;
        private String mediaKind;
        private Map<String, String> metadata;
        private String prefix;
        private Boolean transcriptOnly;

        private Builder(String source, String assetId) {
            this.source = source;
            this.assetId = assetId;
        }

        public Builder url(String url)                          { this.url = url; return this; }
        public Builder s3Key(String s3Key)                      { this.s3Key = s3Key; return this; }
        public Builder assetClass(String assetClass)             { this.assetClass = assetClass; return this; }
        public Builder projectId(String projectId)               { this.projectId = projectId; return this; }
        public Builder mediaKind(String mediaKind)               { this.mediaKind = mediaKind; return this; }
        /** @deprecated use {@link #mediaKind(String)} */
        @Deprecated
        public Builder mediaType(String mediaType)               { this.mediaKind = mediaType; return this; }
        public Builder metadata(Map<String, String> metadata)    { this.metadata = metadata; return this; }
        public Builder transcriptPrefix(String prefix) {
            this.prefix = prefix;
            this.transcriptOnly = true;
            return this;
        }

        public IngestRequest build() { return new IngestRequest(this); }
    }
}
