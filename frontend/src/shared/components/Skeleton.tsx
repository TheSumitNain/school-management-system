import type { CSSProperties } from "react";

type SkeletonProps = {
  width?: CSSProperties["width"];
  height?: CSSProperties["height"];
  radius?: CSSProperties["borderRadius"];
  className?: string;
};

export function Skeleton({ width = "100%", height = 14, radius = 8, className = "" }: SkeletonProps) {
  return (
    <span
      aria-hidden="true"
      className={`skeleton ${className}`}
      style={{ width, height, borderRadius: radius }}
    />
  );
}

export function DashboardSkeleton() {
  return (
    <section className="dashboard-skeleton" aria-label="Loading dashboard" aria-busy="true">
      <div className="skeleton-heading">
        <div><Skeleton width={92} height={10} /><Skeleton width={260} height={28} /><Skeleton width={210} height={12} /></div>
        <Skeleton width={128} height={36} />
      </div>
      <div className="skeleton-stats">
        {Array.from({ length: 4 }, (_, index) => <div className="skeleton-card" key={index}><Skeleton width="52%" /><Skeleton width="32%" height={28} /><Skeleton width="68%" height={10} /></div>)}
      </div>
      <div className="skeleton-feature"><Skeleton width={150} height={10} /><Skeleton width="48%" height={22} /><Skeleton width="70%" height={12} /></div>
    </section>
  );
}
