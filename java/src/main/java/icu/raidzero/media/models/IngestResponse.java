package icu.raidzero.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class IngestResponse {

    @JsonProperty("asset_id") private String assetId;
    @JsonProperty("task_id")  private String taskId;
    private String status;

    public String getAssetId() { return assetId; }
    public String getTaskId()  { return taskId; }
    public String getStatus()  { return status; }
}
