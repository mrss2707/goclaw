import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Users, Search, RefreshCw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useDeferredLoading } from "@/hooks/use-deferred-loading";
import { useMinLoading } from "@/hooks/use-min-loading";
import { useHttp } from "@/hooks/use-ws";
import { toast } from "@/stores/use-toast-store";

interface UserItem {
  id: string;
  email: string;
  google_id: string | null;
  display_name: string;
  avatar_url: string | null;
  email_verified: boolean;
  locale: string;
  created_at: string;
}

interface UsersResponse {
  users: UserItem[];
  total_count: number;
  page: number;
  page_size: number;
}

export function AdminUsersPage() {
  const { t } = useTranslation("admin-users");
  const http = useHttp();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<UserItem | null>(null);
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const spinning = useMinLoading(refreshing);
  const showSkeleton = useDeferredLoading(loading && users.length === 0);

  const fetchUsers = useCallback(async (p: number) => {
    try {
      const params: Record<string, string> = { page: String(p), page_size: String(pageSize) };
      if (search) params.search = search;
      const res = await http.get<UsersResponse>("/v1/admin/users", params);
      setUsers(res.users || []);
      setTotal(res.total_count);
      setPage(p);
    } catch {
      toast.error(t("errors.loadFailed"));
    }
  }, [search, pageSize]);

  useEffect(() => {
    setLoading(true);
    fetchUsers(1).finally(() => setLoading(false));
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchUsers(page).finally(() => setRefreshing(false));
  };

  const handleSearch = async () => {
    setLoading(true);
    await fetchUsers(1).finally(() => setLoading(false));
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await http.delete(`/v1/admin/users/${deleteTarget.id}`);
      toast.success(t("delete.success"));
      setDeleteOpen(false);
      setDeleteTarget(null);
      await handleRefresh();
    } catch {
      toast.error(t("delete.errors.failed"));
    } finally {
      setDeleting(false);
    }
  };

  if (showSkeleton) {
    return (
      <div className="p-4 sm:p-6 pb-10">
        <PageHeader title={t("title")} />
        <div className="mt-6">
          <TableSkeleton />
        </div>
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6 pb-10">
      <PageHeader title={t("title")} />

      <div className="mt-6 space-y-4">
        {/* Search bar */}
        <div className="flex items-center justify-between gap-2">
          <Button variant="outline" size="sm" onClick={handleRefresh} disabled={spinning}>
            <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${spinning ? "animate-spin" : ""}`} />
            {spinning ? "..." : ""}
          </Button>
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("search")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            />
          </div>
          <Button variant="secondary" onClick={handleSearch}>
            <Search className="h-4 w-4" />
          </Button>
        </div>

        {users.length === 0 ? (
          <EmptyState
            icon={Users}
            title={t("noUsers")}
          />
        ) : (
          <>
            <div className="overflow-x-auto rounded-md border">
              <table className="min-w-[600px] w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="px-4 py-3 text-left font-medium">{t("table.email")}</th>
                    <th className="px-4 py-3 text-left font-medium">{t("table.displayName")}</th>
                    <th className="px-4 py-3 text-left font-medium">{t("table.googleLinked")}</th>
                    <th className="px-4 py-3 text-left font-medium">{t("table.createdAt")}</th>
                    <th className="px-4 py-3 text-right font-medium">{t("table.actions")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {users.map((u) => (
                    <tr key={u.id} className="hover:bg-muted/30">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <span>{u.email}</span>
                          {u.email_verified && (
                            <Badge variant="outline" className="text-xs">
                              {t("verified")}
                            </Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {u.display_name || "-"}
                      </td>
                      <td className="px-4 py-3">
                        {u.google_id ? (
                          <span className="text-green-600 font-medium">&#x2713;</span>
                        ) : (
                          <span className="text-muted-foreground">&#x2717;</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setDeleteTarget(u);
                            setDeleteOpen(true);
                          }}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">
                  {users.length} of {total} users
                </span>
                <div className="flex gap-1">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => fetchUsers(page - 1)}
                  >
                    Prev
                  </Button>
                  <span className="flex items-center px-2 text-sm text-muted-foreground">
                    {page} / {totalPages}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => fetchUsers(page + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Delete confirmation dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="max-sm:inset-0 sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("delete.title")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("delete.confirm")}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleting}>
              {deleting ? "..." : t("delete.button")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
