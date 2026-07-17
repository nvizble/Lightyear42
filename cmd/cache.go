package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/nvizble/Lightyear42/internal/cache"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/spf13/cobra"
)

// cacheDBFile is the SQLite database file name inside the cache directory.
const cacheDBFile = "cache.db"

// cacheDBPath returns the full path of the local cache database.
func cacheDBPath() (string, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.CacheDir, cacheDBFile), nil
}

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Gerencia o cache local de respostas da API",
		Long: `O cache local guarda respostas da API da 42 em SQLite com TTL,
reduzindo chamadas repetidas e o consumo do rate limit.`,
	}

	cmd.AddCommand(newCacheClearCmd())

	return cmd
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove todas as entradas do cache",
		Long:  "Apaga todas as respostas da API armazenadas localmente. A próxima consulta busca dados frescos.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dbPath, err := cacheDBPath()
			if err != nil {
				return err
			}

			store, err := cache.Open(dbPath)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := services.NewCacheService(store).Clear(); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Cache limpo.")
			return nil
		},
	}
}
