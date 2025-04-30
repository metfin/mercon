import { Database, type Statement } from "bun:sqlite";
import type { MeteoraDlmmInstruction } from "../services/MeteoraParser";
import type {
  MeteoraDlmmPairData,
  MeteoraPositionTransactions,
  MeteoraDlmmDbTransactions,
} from "../types/meteora";
import type { TokenMeta } from "../types/tokens";
import type { DatabaseInterface } from "./DatabaseInterface";
import fs from "node:fs/promises";
import path from "node:path";

interface PreparedStatements {
  addInstruction: Statement;
  addTransfer: Statement;
  addPair: Statement;
  addToken: Statement;
  addUsdX: Statement;
  addUsdY: Statement;
  fillMissingUsd: Statement;
  setOldestSignature: Statement;
  markComplete: Statement;
  getAllTransactions: Statement;
}

interface QueryResult {
  [key: string]: unknown;
}

/**
 * SQLite implementation of the database interface using Bun's built-in SQLite driver
 */
export class SqliteDatabase implements DatabaseInterface {
  private db: Database | null = null;
  private statements: PreparedStatements | null = null;
  private dbPath: string;
  private savePromise: Promise<void> | null = null;
  private queue: Array<() => Promise<unknown>> = [];
  private processing = false;

  /**
   * Create a new SQLite database
   * @param dbPath Path to store the database file
   */
  constructor(dbPath: string) {
    this.dbPath = dbPath;
  }

  /**
   * Initialize the database
   */
  async initialize(): Promise<void> {
    try {
      // Ensure directory exists
      await fs.mkdir(path.dirname(this.dbPath), { recursive: true });

      // Create or open the database
      const exists = await this.fileExists(this.dbPath);
      this.db = new Database(this.dbPath, { create: !exists, readwrite: true });

      if (!exists) {
        await this.createTables();
        await this.addInitialData();
      }

      this.prepareStatements();
    } catch (error) {
      console.error("Error initializing database:", error);
      throw error;
    }
  }

  /**
   * Check if a file exists
   */
  private async fileExists(filePath: string): Promise<boolean> {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Close the database connection
   */
  async close(): Promise<void> {
    if (this.db) {
      await this.waitForSave();
      this.db.close();
      this.db = null;
      this.statements = null;
    }
  }

  /**
   * Save the database state
   * SQLite saves automatically so this is a no-op
   * But we keep it for API compatibility
   */
  async save(): Promise<void> {
    // Bun's SQLite implementation saves automatically
    await this.processQueue();
  }

  /**
   * Create database tables
   */
  private async createTables(): Promise<void> {
    if (!this.db) throw new Error("Database not initialized");

    this.db.exec(`
      -- Instructions
      CREATE TABLE IF NOT EXISTS instructions (
        signature TEXT NOT NULL,
        slot INTEGER NOT NULL,
        is_hawksight INTEGER NOT NULL,
        block_time INTEGER NOT NULL,
        instruction_name TEXT NOT NULL,
        instruction_type TEXT NOT NULL,
        position_address TEXT NOT NULL,
        pair_address TEXT NOT NULL,
        owner_address TEXT NOT NULL,
        active_bin_id INTEGER,
        removal_bps INTEGER
      );
      CREATE UNIQUE INDEX IF NOT EXISTS idx_instructions_signature_instruction_name_position_address
      ON instructions (
        signature, 
        instruction_name, 
        position_address
      );
      CREATE INDEX IF NOT EXISTS instructions_position_address ON instructions (position_address);
      CREATE INDEX IF NOT EXISTS instructions_block_time ON instructions (block_time);
      CREATE INDEX IF NOT EXISTS instructions_signature ON instructions (signature);

      -- Token Transfers
      CREATE TABLE IF NOT EXISTS token_transfers (
        signature TEXT NOT NULL,
        instruction_name TEXT NOT NULL,
        position_address TEXT NOT NULL,
        mint TEXT NOT NULL,
        amount REAL NOT NULL,
        usd_load_attempted NUMERIC DEFAULT (0) NOT NULL, 
        usd_amount REAL,
        FOREIGN KEY (
          signature, 
          instruction_name, 
          position_address
        ) REFERENCES instructions (
          signature, 
          instruction_name, 
          position_address
        ) ON DELETE CASCADE
      );
      CREATE UNIQUE INDEX IF NOT EXISTS idx_token_transfers_signature_instruction_name_position_address_mint
      ON token_transfers (
        signature, 
        instruction_name, 
        position_address, 
        mint
      );
      CREATE INDEX IF NOT EXISTS token_transfers_position_address ON token_transfers (position_address);

      -- DLMM Pairs
      CREATE TABLE IF NOT EXISTS dlmm_pairs (
        pair_address TEXT NOT NULL,
        name TEXT NOT NULL,
        mint_x TEXT NOT NULL,
        mint_y TEXT NOT NULL,
        bin_step INTEGER NOT NULL,
        base_fee_bps INTEGER NOT NULL
      );
      CREATE UNIQUE INDEX IF NOT EXISTS dlmm_pairs_pair_address
      ON dlmm_pairs (pair_address);

      -- Tokens
      CREATE TABLE IF NOT EXISTS tokens (
        address TEXT NOT NULL,
        name TEXT,
        symbol TEXT,
        decimals INTEGER NOT NULL,
        logo TEXT
      );
      CREATE UNIQUE INDEX IF NOT EXISTS tokens_address
      ON tokens (address);

      -- Quote Tokens
      CREATE TABLE IF NOT EXISTS quote_tokens (
        priority INTEGER NOT NULL,
        mint TEXT NOT NULL
      );
      CREATE UNIQUE INDEX IF NOT EXISTS quote_tokens_priority
      ON quote_tokens (priority);
      CREATE UNIQUE INDEX IF NOT EXISTS quote_tokens_mint
      ON quote_tokens (mint);

      -- Completed Accounts
      CREATE TABLE IF NOT EXISTS completed_accounts (
        account_address TEXT NOT NULL,
        completed INTEGER DEFAULT (0) NOT NULL, 
        oldest_block_time INTEGER, 
        oldest_signature TEXT,
        CONSTRAINT completed_accounts_account_address PRIMARY KEY (account_address)
      );

      -- Instruction Types
      CREATE TABLE IF NOT EXISTS instruction_types (
        priority INTEGER NOT NULL,
        instruction_type INTEGER NOT NULL
      );
      CREATE UNIQUE INDEX IF NOT EXISTS instruction_types_priority
      ON instruction_types (priority);
      CREATE UNIQUE INDEX IF NOT EXISTS instruction_types_instruction_type
      ON instruction_types (instruction_type);
    `);

    // Create views
    await this.createViews();
  }

  /**
   * Add initial reference data
   */
  private async addInitialData(): Promise<void> {
    if (!this.db) throw new Error("Database not initialized");

    this.db.exec(`
      INSERT INTO quote_tokens (priority,mint) VALUES
        (1,'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v'),
        (2,'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB'),
        (3,'So11111111111111111111111111111111111111112')
      ON CONFLICT DO NOTHING
    `);

    this.db.exec(`
      INSERT INTO instruction_types (priority,instruction_type) VALUES
        (1,'open'),
        (2,'add'),
        (3,'claim'),
        (4,'remove'),
        (5,'close')
      ON CONFLICT DO NOTHING
    `);
  }

  /**
   * Create database views
   */
  private async createViews(): Promise<void> {
    if (!this.db) throw new Error("Database not initialized");

    // Missing Pairs view
    this.db.exec(`
      CREATE VIEW IF NOT EXISTS v_missing_pairs AS
      SELECT DISTINCT 
        i.pair_address
      FROM
        instructions i 
        LEFT JOIN dlmm_pairs p ON
          i.pair_address = p.pair_address 
      WHERE 
        p.pair_address IS NULL;
    `);

    // Missing Tokens view
    this.db.exec(`
      CREATE VIEW IF NOT EXISTS v_missing_tokens AS
      SELECT DISTINCT address FROM (
        SELECT
          p.mint_x address
        FROM
          instructions i 
          JOIN dlmm_pairs p ON
            i.pair_address = p.pair_address 
          LEFT JOIN tokens x ON
            p.mint_x  = x.address 
        WHERE 
          x.address IS NULL
        UNION
        SELECT 
          p.mint_y
        FROM
          instructions i 
          JOIN dlmm_pairs p ON
            i.pair_address = p.pair_address 
          LEFT JOIN tokens y ON
            p.mint_y  = y.address 
        WHERE 
          y.address IS NULL
      );
    `);

    // Missing USD view
    this.db.exec(`
      CREATE VIEW IF NOT EXISTS v_missing_usd AS
      SELECT 
        position_address
      FROM
        token_transfers
      GROUP BY
        position_address
      HAVING
        SUM(usd_load_attempted) <> COUNT(*);
    `);

    // Transactions view - a simplified version to start
    this.db.exec(`
      CREATE VIEW IF NOT EXISTS v_transactions AS
      SELECT
        i.block_time,
        i.is_hawksight,
        i.signature,
        i.position_address,
        i.owner_address,
        i.pair_address,
        p.mint_x AS base_mint,
        x.symbol AS base_symbol,
        x.decimals AS base_decimals,
        x.logo AS base_logo,
        p.mint_y AS quote_mint,
        y.symbol AS quote_symbol,
        y.decimals AS quote_decimals,
        y.logo AS quote_logo,
        0 AS is_inverted,
        CASE WHEN EXISTS (
          SELECT 1 FROM instructions c 
          WHERE c.position_address = i.position_address 
          AND c.instruction_type = 'close'
        ) THEN 0 ELSE 1 END AS position_is_open,
        CASE WHEN i.instruction_type = 'open' THEN 1 ELSE 0 END AS is_opening_transaction,
        CASE WHEN i.instruction_type = 'close' THEN 1 ELSE 0 END AS is_closing_transaction,
        0 AS price,
        0 AS fee_amount,
        0 AS deposit,
        0 AS withdrawal,
        0 AS usd_fee_amount,
        0 AS usd_deposit,
        0 AS usd_withdrawal
      FROM
        instructions i
        LEFT JOIN dlmm_pairs p ON i.pair_address = p.pair_address
        LEFT JOIN tokens x ON p.mint_x = x.address
        LEFT JOIN tokens y ON p.mint_y = y.address
      ORDER BY
        i.block_time DESC;
    `);
  }

  /**
   * Prepare database statements
   */
  private prepareStatements(): void {
    if (!this.db) throw new Error("Database not initialized");

    this.statements = {
      addInstruction: this.db.prepare(`
        INSERT INTO instructions(
          signature, 
          slot, 
          block_time, 
          is_hawksight,
          instruction_name,
          instruction_type,
          position_address,
          pair_address,
          owner_address,
          active_bin_id,
          removal_bps
        )
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT DO NOTHING
      `),

      addTransfer: this.db.prepare(`
        INSERT INTO token_transfers(
          signature,
          instruction_name,
          position_address,
          mint,
          amount
        )
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT DO NOTHING
      `),

      addPair: this.db.prepare(`
        INSERT INTO dlmm_pairs(
          pair_address,
          name,
          mint_x,
          mint_y,
          bin_step,
          base_fee_bps
        )
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT DO NOTHING
      `),

      addToken: this.db.prepare(`
        INSERT INTO tokens(
          address,
          name,
          symbol,
          decimals,
          logo
        )
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT DO NOTHING
      `),

      addUsdX: this.db.prepare(`
        UPDATE token_transfers
        SET 
          usd_load_attempted = 1,
          usd_amount = ?
        WHERE EXISTS (
          SELECT 
            1
          FROM
            token_transfers t
            JOIN instructions i ON
              i.signature = t.signature
              AND i.instruction_name = t.instruction_name
              AND i.position_address = t.position_address
            JOIN dlmm_pairs p ON
              i.pair_address = p.pair_address
          WHERE
            t.signature = ?
            AND token_transfers.signature = t.signature
            AND token_transfers.instruction_name = t.instruction_name
            AND token_transfers.position_address = ?
            AND token_transfers.mint = p.mint_x
            AND i.instruction_type = ?
        )
      `),

      addUsdY: this.db.prepare(`
        UPDATE token_transfers
        SET 
          usd_load_attempted = 1,
          usd_amount = ?
        WHERE EXISTS (
          SELECT 
            1
          FROM
            token_transfers t
            JOIN instructions i ON
              i.signature = t.signature
              AND i.instruction_name = t.instruction_name
              AND i.position_address = t.position_address
            JOIN dlmm_pairs p ON
              i.pair_address = p.pair_address
          WHERE
            t.signature = ?
            AND token_transfers.signature = t.signature
            AND token_transfers.instruction_name = t.instruction_name
            AND token_transfers.position_address = ?
            AND token_transfers.mint = p.mint_y
            AND i.instruction_type = ?
        )
      `),

      fillMissingUsd: this.db.prepare(`
        UPDATE token_transfers
        SET 
          usd_load_attempted = 1
        WHERE EXISTS (
          SELECT 
            1
          FROM
            token_transfers t
          WHERE
            t.position_address = token_transfers.position_address
            AND token_transfers.usd_load_attempted = 0
            AND t.position_address = ?
        )
      `),

      setOldestSignature: this.db.prepare(`
        INSERT INTO completed_accounts (account_address, oldest_block_time, oldest_signature)
        VALUES (?, ?, ?)
        ON CONFLICT DO UPDATE
        SET 
          oldest_block_time = excluded.oldest_block_time,
          oldest_signature = excluded.oldest_signature
      `),

      markComplete: this.db.prepare(`
        INSERT INTO completed_accounts (account_address, completed)
        VALUES (?, 1)
        ON CONFLICT DO UPDATE
        SET 
          completed = 1
      `),

      getAllTransactions: this.db.prepare(`
        SELECT * FROM v_transactions
      `),
    };
  }

  /**
   * Queue a database operation
   */
  private async queueOperation<T>(operation: () => Promise<T>): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      this.queue.push(async () => {
        try {
          const result = await operation();
          resolve(result);
        } catch (error) {
          reject(error);
        }
      });

      void this.processQueue();
    });
  }

  /**
   * Process the operation queue
   */
  private async processQueue(): Promise<void> {
    if (this.processing || this.queue.length === 0) {
      return;
    }

    this.processing = true;
    this.savePromise = (async () => {
      while (this.queue.length > 0) {
        const operation = this.queue.shift();
        if (operation) {
          await operation();
        }
      }
    })();

    await this.savePromise;
    this.processing = false;
    this.savePromise = null;
  }

  /**
   * Wait for pending operations to complete
   */
  async waitForSave(): Promise<void> {
    if (this.savePromise) {
      await this.savePromise;
    }
  }

  /**
   * Add an instruction to the database
   */
  async addInstruction(instruction: MeteoraDlmmInstruction): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      const {
        signature,
        slot,
        blockTime,
        // Convert isHawksight if it doesn't exist
        accounts,
        activeBinId,
        removalBps,
        instructionName,
        instructionType,
      } = instruction;
      const { position, lbPair, sender } = accounts;
      // Handle isHawksight property which might not exist
      const isHawksight =
        "isHawksight" in instruction ? instruction.isHawksight : false;

      this.statements.addInstruction.run(
        signature,
        slot,
        blockTime,
        isHawksight ? 1 : 0,
        instructionName,
        instructionType,
        position,
        lbPair,
        sender,
        activeBinId,
        removalBps
      );
    });
  }

  /**
   * Add token transfers for an instruction
   */
  async addTransfers(instruction: MeteoraDlmmInstruction): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      const { signature, instructionName, accounts, tokenTransfers } =
        instruction;
      const { position } = accounts;

      for (const transfer of tokenTransfers) {
        const { mint, amount } = transfer;
        this.statements.addTransfer.run(
          signature,
          instructionName,
          position,
          mint,
          amount
        );
      }
    });
  }

  /**
   * Add a DLMM pair to the database
   */
  async addPair(pair: MeteoraDlmmPairData): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      const { lbPair, name, mintX, mintY, binStep, baseFeeBps } = pair;

      this.statements.addPair.run(
        lbPair,
        name,
        mintX,
        mintY,
        binStep,
        baseFeeBps
      );
    });
  }

  /**
   * Get the LB pair for a position
   */
  async getLbPair(positionAddress: string): Promise<string | undefined> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db
        .query(`
        SELECT 
          pair_address
        FROM
          instructions
        WHERE
          position_address = ?
        LIMIT 1
      `)
        .get(positionAddress) as QueryResult | null;

      return result ? (result["pair_address"] as string) : undefined;
    });
  }

  /**
   * Get missing pairs from the database
   */
  async getMissingPairs(): Promise<string[]> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db.query("SELECT * FROM v_missing_pairs").all();
      return result.map(
        (row) => (row as QueryResult)["pair_address"] as string
      );
    });
  }

  /**
   * Add a token to the database
   */
  async addToken(token: TokenMeta): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      const { address, name, symbol, decimals, logoURI } = token;

      this.statements.addToken.run(
        address,
        name || null,
        symbol || null,
        decimals,
        logoURI || null
      );
    });
  }

  /**
   * Get missing tokens from the database
   */
  async getMissingTokens(): Promise<string[]> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db.query("SELECT * FROM v_missing_tokens").all();
      return result.map((row) => (row as QueryResult)["address"] as string);
    });
  }

  /**
   * Add USD transaction data
   */
  async addUsdTransactions(
    positionAddress: string,
    transactions: MeteoraPositionTransactions
  ): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      // Add deposits
      for (const deposit of transactions.deposits) {
        const { tx_id, token_x_usd_amount, token_y_usd_amount } = deposit;
        this.statements.addUsdX.run(
          token_x_usd_amount,
          tx_id,
          positionAddress,
          "add"
        );
        this.statements.addUsdY.run(
          token_y_usd_amount,
          tx_id,
          positionAddress,
          "add"
        );
      }

      // Add withdrawals
      for (const withdrawal of transactions.withdrawals) {
        const { tx_id, token_x_usd_amount, token_y_usd_amount } = withdrawal;
        this.statements.addUsdX.run(
          token_x_usd_amount,
          tx_id,
          positionAddress,
          "remove"
        );
        this.statements.addUsdY.run(
          token_y_usd_amount,
          tx_id,
          positionAddress,
          "remove"
        );
      }

      // Add fees
      for (const fee of transactions.fees) {
        const { tx_id, token_x_usd_amount, token_y_usd_amount } = fee;
        this.statements.addUsdX.run(
          token_x_usd_amount,
          tx_id,
          positionAddress,
          "claim"
        );
        this.statements.addUsdY.run(
          token_y_usd_amount,
          tx_id,
          positionAddress,
          "claim"
        );
      }

      // Mark any remaining transfers as having USD loaded
      this.statements.fillMissingUsd.run(positionAddress);
    });
  }

  /**
   * Get positions missing USD data
   */
  async getMissingUsd(): Promise<string[]> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db.query("SELECT * FROM v_missing_usd").all();
      return result.map(
        (row) => (row as QueryResult)["position_address"] as string
      );
    });
  }

  /**
   * Set the oldest signature for an account
   */
  async setOldestSignature(
    accountAddress: string,
    oldestBlockTime: number,
    oldestSignature: string
  ): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      this.statements.setOldestSignature.run(
        accountAddress,
        oldestBlockTime,
        oldestSignature
      );
    });
  }

  /**
   * Mark an account as completed
   */
  async markComplete(accountAddress: string): Promise<void> {
    return this.queueOperation(async () => {
      if (!this.db || !this.statements)
        throw new Error("Database not initialized");

      this.statements.markComplete.run(accountAddress);
    });
  }

  /**
   * Check if an account is complete
   */
  async isComplete(accountAddress: string): Promise<boolean> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db
        .query(`
        SELECT 
          account_address
        FROM
          completed_accounts
        WHERE
          account_address = ?
          AND completed
      `)
        .get(accountAddress) as QueryResult | null;

      return result !== null;
    });
  }

  /**
   * Get the most recent signature for an owner
   */
  async getMostRecentSignature(
    ownerAddress: string
  ): Promise<string | undefined> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db
        .query(`
        SELECT 
          signature
        FROM
          instructions
        WHERE
          owner_address = ?
        ORDER BY
          block_time DESC
        LIMIT 1
      `)
        .get(ownerAddress) as QueryResult | null;

      return result ? (result["signature"] as string) : undefined;
    });
  }

  /**
   * Get the oldest signature for an owner
   */
  async getOldestSignature(ownerAddress: string): Promise<string | undefined> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db
        .query(`
        WITH signatures AS (
          SELECT 
            block_time, signature
          FROM
            instructions
          WHERE
            owner_address = ?
          UNION
          SELECT
            oldest_block_time, oldest_signature
          FROM
            completed_accounts
          WHERE
            account_address = ?
        )
        SELECT
          signature
        FROM
          signatures
        ORDER BY
          block_time 
        LIMIT 1
      `)
        .get(ownerAddress, ownerAddress) as QueryResult | null;

      return result ? (result["signature"] as string) : undefined;
    });
  }

  /**
   * Add a transaction to the database
   */
  async addTransaction(transaction: {
    signature: string;
    owner: string;
    timestamp: string;
    slot: number;
  }): Promise<void> {
    if (!this.db) throw new Error("Database not initialized");

    // Store a reference to the db so TypeScript knows it's not null in the callback
    const db = this.db;

    return this.queueOperation(async () => {
      // Convert timestamp to a block_time (Unix timestamp)
      const blockTime = Math.floor(
        new Date(transaction.timestamp).getTime() / 1000
      );

      // Add a basic transaction record
      // This will be enriched later when processing instructions
      db.exec(`
        INSERT OR IGNORE INTO instructions (
          signature,
          slot,
          is_hawksight,
          block_time,
          instruction_name,
          instruction_type,
          position_address,
          pair_address,
          owner_address
        ) VALUES (
          '${transaction.signature}',
          ${transaction.slot},
          0,
          ${blockTime},
          'transaction_record',
          'transaction',
          '${transaction.owner}',
          '',
          '${transaction.owner}'
        )
      `);
    });
  }

  /**
   * Get all transactions from the database
   */
  async getAllTransactions(): Promise<MeteoraDlmmDbTransactions[]> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db.query("SELECT * FROM v_transactions").all();
      return result as MeteoraDlmmDbTransactions[];
    });
  }

  /**
   * Get transactions for a specific owner
   */
  async getOwnerTransactions(
    ownerAddress: string
  ): Promise<MeteoraDlmmDbTransactions[]> {
    return this.queueOperation(async () => {
      if (!this.db) throw new Error("Database not initialized");

      const result = this.db
        .query(`
        SELECT * FROM v_transactions 
        WHERE owner_address = ?
      `)
        .all(ownerAddress);

      return result as MeteoraDlmmDbTransactions[];
    });
  }
}
