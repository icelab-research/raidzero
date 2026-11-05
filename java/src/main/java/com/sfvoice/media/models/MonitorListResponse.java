package com.sfvoice.media.models;

import java.util.List;

public class MonitorListResponse {

    private List<Monitor> items;
    private long total;

    public List<Monitor> getItems() { return items; }
    public long getTotal()          { return total; }
}
