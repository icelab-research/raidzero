package com.sfvoice.media.models;

import java.util.List;

public class MonitorEventListResponse {

    private List<MonitorEvent> items;
    private long total;

    public List<MonitorEvent> getItems() { return items; }
    public long getTotal()               { return total; }
}
