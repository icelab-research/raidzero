package icu.raidzero.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Task {

    @JsonProperty("task_id")      private String taskId;
    @JsonProperty("asset_id")     private String assetId;
    private String status;
    private String error;
    @JsonProperty("created_at")   private String createdAt;
    @JsonProperty("completed_at") private String completedAt;

    public String getTaskId()      { return taskId; }
    public String getAssetId()     { return assetId; }
    public String getStatus()      { return status; }
    public String getError()       { return error; }
    public String getCreatedAt()   { return createdAt; }
    public String getCompletedAt() { return completedAt; }

    public boolean isTerminal() {
        return "ready".equals(status) || "failed".equals(status);
    }
}
