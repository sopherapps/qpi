import { type PropsWithChildren } from "react";

export function InnerTabBar({ onTabClick, currentTab }: Props) {
  return (
    <div className="flex border-b border-gray-200 dark:border-zinc-800 mb-6">
      <_TabBtn
        value="users"
        onClick={onTabClick}
        isActive={currentTab === "users"}
      >
        User Time Allocations
      </_TabBtn>
      <_TabBtn
        value="announcements"
        onClick={onTabClick}
        isActive={currentTab === "announcements"}
      >
        Broadcast Announcement
      </_TabBtn>
      <_TabBtn
        value="requests"
        onClick={onTabClick}
        isActive={currentTab === "requests"}
      >
        Time Requests
      </_TabBtn>
      <_TabBtn
        value="appearance"
        onClick={onTabClick}
        isActive={currentTab === "appearance"}
      >
        Appearance
      </_TabBtn>
    </div>
  );
}

function _TabBtn({
  onClick,
  children,
  value,
  isActive,
}: PropsWithChildren<_TabBtnProps>) {
  return (
    <button
      onClick={() => onClick(value)}
      className={`px-4 py-2 font-geist text-sm transition-all -mb-[1px] ${
        isActive
          ? "text-gray-900 dark:text-white border-b-2 border-white font-medium"
          : "text-gray-400 dark:text-zinc-500 hover:text-gray-600 dark:text-zinc-300"
      }`}
    >
      {children}
    </button>
  );
}

interface _TabBtnProps {
  onClick: (tab: string) => void;
  value: string;
  isActive: boolean;
}

interface Props {
  onTabClick: (tab: string) => void;
  currentTab: string;
}
