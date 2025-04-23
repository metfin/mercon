import type { DatabaseInterface } from "./DatabaseInterface";
import { SqliteDatabase } from "./SqliteDatabase";
import path from "node:path";
import os from "node:os";

export enum DatabaseType {
	SQLITE = "sqlite",
	MEMORY = "memory",
	// Future support for other database types
	// POSTGRES = 'postgres',
}

export interface DatabaseConfig {
	type: DatabaseType;
	path?: string; // For file-based databases like SQLite
	// Connection info for remote databases
	// host?: string;
	// port?: number;
	// user?: string;
	// password?: string;
	// database?: string;
}

/**
 * Create database instances based on configuration
 */
export const createDatabase = async (
	config: DatabaseConfig,
): Promise<DatabaseInterface> => {
	switch (config.type) {
		case DatabaseType.SQLITE: {
			const dbPath =
				config.path || path.join(os.homedir(), ".mercon", "meteora.db");
			const db = new SqliteDatabase(dbPath);
			await db.initialize();
			return db;
		}

		case DatabaseType.MEMORY: {
			// In-memory SQLite database
			const memDb = new SqliteDatabase(":memory:");
			await memDb.initialize();
			return memDb;
		}

		// Future database types
		// case DatabaseType.POSTGRES:
		//   return new PostgresDatabase(config);

		default:
			throw new Error(`Unsupported database type: ${config.type}`);
	}
};
