package icu.raidzero.media.models;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class AssetListResponse {

    private List<Asset> items;
    @JsonProperty("page_info") private PageInfo pageInfo;

    public List<Asset> getItems()  { return items; }
    public PageInfo getPageInfo()  { return pageInfo; }
}
