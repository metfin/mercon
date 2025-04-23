/**
 * Data downloaded from Meteora
 */
export interface DownloadedData {
	account: string;
	pairs: string[];
	positions: string[];
	transactions: number;
	startTime: Date;
	endTime: Date;
}
