package com.sfvoice.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;

public class Asset {

    @JsonProperty("asset_id") private String assetId;
    private String filename;
    @JsonProperty("project_id") private String projectId;
    @JsonProperty("asset_class") private String assetClass;
    private String status;
    @JsonProperty("media_type")  private String mediaType;
    @JsonProperty("source_type") private String sourceType;
    private java.util.List<String> types;
    private Map<String, Object> metadata;
    @JsonProperty("duration_ms") private Long durationMs;
    @JsonProperty("thumbnail_url") private String thumbnailUrl;
    @JsonProperty("thumbnail_status") private String thumbnailStatus;
    @JsonProperty("created_at")  private String createdAt;
    @JsonProperty("updated_at")  private String updatedAt;

    public String getAssetId()               { return assetId; }
    /** @deprecated use {@link #getAssetId()} */
    @Deprecated
    public String getId()                    { return assetId; }
    public String getFilename()              { return filename; }
    public String getProjectId()             { return projectId; }
    public String getAssetClass()            { return assetClass; }
    public String getStatus()                { return status; }
    public String getMediaType()             { return mediaType; }
    public String getSourceType()            { return sourceType; }
    public java.util.List<String> getTypes() { return types; }
    public Map<String, String> getMetadata() {
        Map<String, String> strings = new java.util.HashMap<>();
        if (metadata != null) {
            metadata.forEach((key, value) -> {
                if (value instanceof String text) strings.put(key, text);
            });
        }
        return strings;
    }
    public Map<String, Object> getMetadataValues() { return metadata; }
    public Long getDurationMs()              { return durationMs; }
    public String getThumbnailUrl()          { return thumbnailUrl; }
    public String getThumbnailStatus()       { return thumbnailStatus; }
    public String getCreatedAt()             { return createdAt; }
    public String getUpdatedAt()             { return updatedAt; }
}
