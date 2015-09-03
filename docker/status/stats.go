package main

type Stats struct {
	Read    string `json:"read"`
	Network struct {
		RxBytes   int `json:"rx_bytes"`
		RxPackets int `json:"rx_packets"`
		RxErrors  int `json:"rx_errors"`
		RxDropped int `json:"rx_dropped"`
		TxBytes   int `json:"tx_bytes"`
		TxPackets int `json:"tx_packets"`
		TxErrors  int `json:"tx_errors"`
		TxDropped int `json:"tx_dropped"`
	} `json:"network"`
	PrecpuStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	MemoryStats struct {
		Usage    int `json:"usage"`
		MaxUsage int `json:"max_usage"`
		Stats    struct {
			ActiveAnon              int   `json:"active_anon"`
			ActiveFile              int   `json:"active_file"`
			Cache                   int   `json:"cache"`
			HierarchicalMemoryLimit int64 `json:"hierarchical_memory_limit"`
			HierarchicalMemswLimit  int64 `json:"hierarchical_memsw_limit"`
			InactiveAnon            int   `json:"inactive_anon"`
			InactiveFile            int   `json:"inactive_file"`
			MappedFile              int   `json:"mapped_file"`
			Pgfault                 int   `json:"pgfault"`
			Pgmajfault              int   `json:"pgmajfault"`
			Pgpgin                  int   `json:"pgpgin"`
			Pgpgout                 int   `json:"pgpgout"`
			RecentRotatedAnon       int   `json:"recent_rotated_anon"`
			RecentRotatedFile       int   `json:"recent_rotated_file"`
			RecentScannedAnon       int   `json:"recent_scanned_anon"`
			RecentScannedFile       int   `json:"recent_scanned_file"`
			Rss                     int   `json:"rss"`
			RssHuge                 int   `json:"rss_huge"`
			Swap                    int   `json:"swap"`
			TotalActiveAnon         int   `json:"total_active_anon"`
			TotalActiveFile         int   `json:"total_active_file"`
			TotalCache              int   `json:"total_cache"`
			TotalInactiveAnon       int   `json:"total_inactive_anon"`
			TotalInactiveFile       int   `json:"total_inactive_file"`
			TotalMappedFile         int   `json:"total_mapped_file"`
			TotalPgfault            int   `json:"total_pgfault"`
			TotalPgmajfault         int   `json:"total_pgmajfault"`
			TotalPgpgin             int   `json:"total_pgpgin"`
			TotalPgpgout            int   `json:"total_pgpgout"`
			TotalRss                int   `json:"total_rss"`
			TotalRssHuge            int   `json:"total_rss_huge"`
			TotalSwap               int   `json:"total_swap"`
			TotalUnevictable        int   `json:"total_unevictable"`
			TotalWriteback          int   `json:"total_writeback"`
			Unevictable             int   `json:"unevictable"`
			Writeback               int   `json:"writeback"`
		} `json:"stats"`
		Failcnt int   `json:"failcnt"`
		Limit   int64 `json:"limit"`
	} `json:"memory_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_service_bytes_recursive"`
		IoServicedRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_serviced_recursive"`
		IoQueueRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_queue_recursive"`
		IoServiceTimeRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_service_time_recursive"`
		IoWaitTimeRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_wait_time_recursive"`
		IoMergedRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_merged_recursive"`
		IoTimeRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_time_recursive"`
		SectorsRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"sectors_recursive"`
	} `json:"blkio_stats"`
}
