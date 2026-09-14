package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"okf/internal/audit"
	"okf/internal/graph"
	"okf/internal/scanner"
	"okf/pkg/okf"

	"github.com/spf13/cobra"
)

var (
	vaultFlag string
	jsonFlag  bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getVaultPath() string {
	if vaultFlag != "" {
		return vaultFlag
	}
	if env := os.Getenv("OBSIDIAN_VAULT_PATH"); env != "" {
		return env
	}
	return "."
}

func printJSON(cmd *cobra.Command, data interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

var rootCmd = &cobra.Command{
	Use:   "okf",
	Short: "High-Performance CLI for Google Open Knowledge Format (OKF v0.2)",
	Long:  "okf is an ultra-fast, concurrent CLI tool to parse, query, traverse, and audit OKF v0.2 knowledge vaults.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&vaultFlag, "vault", "", "Path to knowledge vault (defaults to $OBSIDIAN_VAULT_PATH or '.')")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in machine-readable JSON format")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(relationsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(exportCmd)
}

// ---------------------------------------------------------------------
// 1. okf list
// ---------------------------------------------------------------------
var (
	listTypeFlag   string
	listStatusFlag string
	listTagFlag    string
	listKindFlag   string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List and filter knowledge documents in the vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPath := getVaultPath()
		catalog, err := scanner.ScanVault(vaultPath, nil)
		if err != nil {
			return err
		}

		var filtered []*okf.OKFDocument
		for _, node := range catalog.Nodes {
			if node.Doc == nil {
				continue
			}
			doc := node.Doc

			// Filter by kind
			if listKindFlag != "" && !strings.EqualFold(string(doc.Kind), listKindFlag) {
				continue
			}
			// Filter by type
			if listTypeFlag != "" && !strings.EqualFold(doc.Type, listTypeFlag) {
				continue
			}
			// Filter by status
			if listStatusFlag != "" && !strings.EqualFold(doc.Status, listStatusFlag) {
				continue
			}
			// Filter by tag
			if listTagFlag != "" {
				hasTag := false
				for _, t := range doc.Tags {
					if strings.EqualFold(t, listTagFlag) {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			filtered = append(filtered, doc)
		}

		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].RelPath < filtered[j].RelPath
		})

		if jsonFlag {
			return printJSON(cmd, filtered)
		}

		out := cmd.OutOrStdout()
		if len(filtered) == 0 {
			fmt.Fprintln(out, "No documents matched the criteria.")
			return nil
		}

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "STEM\tKIND\tTYPE\tSTATUS\tTAGS\tPATH")
		fmt.Fprintln(w, "----\t----\t----\t------\t----\t----")
		for _, d := range filtered {
			tags := strings.Join(d.Tags, ",")
			if tags == "" {
				tags = "-"
			}
			status := d.Status
			if status == "" {
				status = "-"
			}
			cType := d.Type
			if cType == "" {
				cType = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Stem, d.Kind, cType, status, tags, d.RelPath)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().StringVar(&listTypeFlag, "type", "", "Filter by OKF concept type (e.g. concept, standard, research-hub)")
	listCmd.Flags().StringVar(&listStatusFlag, "status", "", "Filter by status (draft, stable, deprecated)")
	listCmd.Flags().StringVar(&listTagFlag, "tag", "", "Filter by tag")
	listCmd.Flags().StringVar(&listKindFlag, "kind", "", "Filter by document kind (concept, index, log)")
}

// ---------------------------------------------------------------------
// 2. okf get <query>
// ---------------------------------------------------------------------
var getCmd = &cobra.Command{
	Use:   "get <query>",
	Short: "Retrieve metadata and details for a specific knowledge node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		vaultPath := getVaultPath()
		catalog, err := scanner.ScanVault(vaultPath, nil)
		if err != nil {
			return err
		}

		node := catalog.FindNode(query)
		if node == nil {
			return fmt.Errorf("node '%s' not found in vault", query)
		}

		if node.Doc == nil {
			return fmt.Errorf("node '%s' has frontmatter parse error: %w", query, node.Err)
		}

		if jsonFlag {
			return printJSON(cmd, node.Doc)
		}

		out := cmd.OutOrStdout()
		doc := node.Doc
		fmt.Fprintf(out, "Title:       %s\n", doc.Title)
		fmt.Fprintf(out, "Stem:        %s\n", doc.Stem)
		fmt.Fprintf(out, "Path:        %s\n", doc.RelPath)
		fmt.Fprintf(out, "Kind:        %s\n", doc.Kind)
		if doc.Type != "" {
			fmt.Fprintf(out, "Type:        %s\n", doc.Type)
		}
		if doc.Status != "" {
			fmt.Fprintf(out, "Status:      %s\n", doc.Status)
		}
		if doc.Description != "" {
			fmt.Fprintf(out, "Description: %s\n", doc.Description)
		}
		if len(doc.Tags) > 0 {
			fmt.Fprintf(out, "Tags:        %s\n", strings.Join(doc.Tags, ", "))
		}
		if doc.StaleAfter != "" {
			fmt.Fprintf(out, "Stale After: %s\n", doc.StaleAfter)
		}
		if doc.Generated != nil {
			fmt.Fprintf(out, "Generated:   By: %s  At: %s\n", doc.Generated.By, doc.Generated.At)
		}
		if len(doc.Verified) > 0 {
			fmt.Fprintln(out, "Verified:")
			for _, v := range doc.Verified {
				fmt.Fprintf(out, "  - By: %s  At: %s\n", v.By, v.At)
			}
		}
		if len(doc.Relations) > 0 {
			fmt.Fprintln(out, "Declared Relations:")
			for _, r := range doc.Relations {
				fmt.Fprintf(out, "  - [%s] -> %s\n", r.Type, r.Target)
			}
		}
		if len(doc.Sources) > 0 {
			fmt.Fprintln(out, "Sources:")
			for _, s := range doc.Sources {
				if s.Title != "" {
					fmt.Fprintf(out, "  - %s (%s)\n", s.Title, s.Resource)
				} else {
					fmt.Fprintf(out, "  - %s\n", s.Resource)
				}
			}
		}
		if len(doc.Attributes) > 0 {
			fmt.Fprintln(out, "Attributes:")
			for k, v := range doc.Attributes {
				fmt.Fprintf(out, "  %s: %v\n", k, v)
			}
		}

		return nil
	},
}

// ---------------------------------------------------------------------
// 3. okf relations <query>
// ---------------------------------------------------------------------
var relationsCmd = &cobra.Command{
	Use:   "relations <query>",
	Short: "Inspect the neighborhood (incoming and outgoing links) of a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		vaultPath := getVaultPath()
		catalog, err := scanner.ScanVault(vaultPath, nil)
		if err != nil {
			return err
		}

		node := catalog.FindNode(query)
		if node == nil {
			return fmt.Errorf("node '%s' not found in vault", query)
		}

		g, err := graph.BuildGraph(catalog, true)
		if err != nil {
			return err
		}

		neigh := g.GetNeighborhood(node)

		if jsonFlag {
			type JSONNeighborhood struct {
				Node              *okf.OKFDocument  `json:"node"`
				DeclaredRelations []graph.GraphEdge `json:"declared_relations"`
				BodyLinks         []graph.GraphEdge `json:"body_links"`
				Backlinks         []graph.GraphEdge `json:"backlinks"`
			}
			return printJSON(cmd, JSONNeighborhood{
				Node:              neigh.Node.Doc,
				DeclaredRelations: neigh.DeclaredRelations,
				BodyLinks:         neigh.BodyLinks,
				Backlinks:         neigh.Backlinks,
			})
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Neighborhood for: %s (%s)\n\n", node.Stem, node.RelPath)

		// 1. Declared Relations
		fmt.Fprintf(out, "Declared Relations (%d):\n", len(neigh.DeclaredRelations))
		if len(neigh.DeclaredRelations) == 0 {
			fmt.Fprintln(out, "  (none)")
		} else {
			for _, e := range neigh.DeclaredRelations {
				status := "RESOLVED"
				targetStr := e.Target
				if !e.Resolved {
					status = "UNRESOLVED"
					targetStr = e.RawTarget
				}
				fmt.Fprintf(out, "  - [%s] -> %s [%s]\n", e.Type, targetStr, status)
			}
		}
		fmt.Fprintln(out)

		// 2. Body Links
		fmt.Fprintf(out, "Outgoing Body Links (%d):\n", len(neigh.BodyLinks))
		if len(neigh.BodyLinks) == 0 {
			fmt.Fprintln(out, "  (none)")
		} else {
			for _, e := range neigh.BodyLinks {
				status := "RESOLVED"
				targetStr := e.Target
				if !e.Resolved {
					status = "UNRESOLVED"
					targetStr = e.RawTarget
				}
				label := ""
				if e.Label != "" && e.Label != targetStr {
					label = fmt.Sprintf(" (text: %q)", e.Label)
				}
				fmt.Fprintf(out, "  - (%s) -> %s%s [%s]\n", e.Syntax, targetStr, label, status)
			}
		}
		fmt.Fprintln(out)

		// 3. Backlinks
		fmt.Fprintf(out, "Incoming Backlinks (%d):\n", len(neigh.Backlinks))
		if len(neigh.Backlinks) == 0 {
			fmt.Fprintln(out, "  (none)")
		} else {
			for _, e := range neigh.Backlinks {
				fmt.Fprintf(out, "  <- %s (via %s [%s])\n", e.Source, e.Syntax, e.Type)
			}
		}

		return nil
	},
}

// ---------------------------------------------------------------------
// 4. okf audit
// ---------------------------------------------------------------------
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit vault health and compliance with OKF v0.2",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPath := getVaultPath()
		catalog, err := scanner.ScanVault(vaultPath, nil)
		if err != nil {
			return err
		}

		g, err := graph.BuildGraph(catalog, true)
		if err != nil {
			return err
		}

		report := audit.RunAudit(g, time.Now())

		if jsonFlag {
			if err := printJSON(cmd, report); err != nil {
				return err
			}
			if !report.Passed {
				return fmt.Errorf("audit failed with %d errors", report.ErrorCount)
			}
			return nil
		}

		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "==================================================")
		fmt.Fprintf(out, "OKF v0.2 Vault Audit Report: %s\n", report.VaultPath)
		fmt.Fprintln(out, "==================================================")
		fmt.Fprintf(out, "Total Files Scanned: %d\n", report.TotalScanned)
		fmt.Fprintf(out, "Errors:              %d\n", report.ErrorCount)
		fmt.Fprintf(out, "Warnings:            %d\n", report.WarningCount)
		fmt.Fprintln(out, "--------------------------------------------------")

		if len(report.Issues) == 0 {
			fmt.Fprintln(out, "SUCCESS: Vault is 100% compliant with OKF v0.2!")
			return nil
		}

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SEVERITY\tFILE\tRULE\tMESSAGE")
		fmt.Fprintln(w, "--------\t----\t----\t-------")
		for _, issue := range report.Issues {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Severity, issue.File, issue.Rule, issue.Message)
		}
		w.Flush()
		fmt.Fprintln(out, "--------------------------------------------------")

		if !report.Passed {
			fmt.Fprintf(out, "FAILED: %d critical errors detected.\n", report.ErrorCount)
			return fmt.Errorf("audit failed with %d errors", report.ErrorCount)
		}

		fmt.Fprintf(out, "PASSED WITH WARNINGS: 0 errors, %d warnings.\n", report.WarningCount)
		return nil
	},
}

// ---------------------------------------------------------------------
// 5. okf export
// ---------------------------------------------------------------------
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the full vault graph as JSON adjacency list",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultPath := getVaultPath()
		catalog, err := scanner.ScanVault(vaultPath, nil)
		if err != nil {
			return err
		}

		g, err := graph.BuildGraph(catalog, true)
		if err != nil {
			return err
		}

		var docs []*okf.OKFDocument
		for _, node := range catalog.Nodes {
			if node.Doc != nil {
				docs = append(docs, node.Doc)
			}
		}

		sort.Slice(docs, func(i, j int) bool {
			return docs[i].RelPath < docs[j].RelPath
		})

		var allEdges []graph.GraphEdge
		for _, edges := range g.ForwardEdges {
			allEdges = append(allEdges, edges...)
		}
		sort.Slice(allEdges, func(i, j int) bool {
			if allEdges[i].Source == allEdges[j].Source {
				return allEdges[i].Target < allEdges[j].Target
			}
			return allEdges[i].Source < allEdges[j].Source
		})

		type FullExport struct {
			VaultPath string             `json:"vault_path"`
			Nodes     []*okf.OKFDocument `json:"nodes"`
			Edges     []graph.GraphEdge  `json:"edges"`
		}

		return printJSON(cmd, FullExport{
			VaultPath: catalog.VaultPath,
			Nodes:     docs,
			Edges:     allEdges,
		})
	},
}
