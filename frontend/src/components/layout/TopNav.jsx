import Icon from "../ui/Icon";

const TopNav = ({ title = "Dashboard" }) => {
    return (
        <header className="flex h-16 flex-shrink-0 items-center justify-between whitespace-nowrap border-b border-slate-200 bg-white px-8 dark:border-slate-800 dark:bg-slate-900">
            <div className="flex items-center gap-4">
                <h2 className="text-lg font-bold text-slate-800 dark:text-white">
                    {title}
                </h2>
            </div>
            <div className="flex flex-1 items-center justify-end gap-4">
                {/* Search */}
                <label className="relative hidden w-full max-w-xs md:block">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 dark:text-slate-500">
                        <Icon name="search" />
                    </span>
                    <input
                        className="form-input h-10 w-full rounded-lg border-slate-200 bg-slate-100 pl-10 text-slate-800 placeholder:text-slate-400 focus:border-primary focus:ring-primary/50 dark:border-slate-700 dark:bg-slate-800 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-primary dark:focus:ring-primary/50"
                        placeholder="Search..."
                        type="search"
                    />
                </label>

                {/* Notifications */}
                <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-slate-100 text-slate-500 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700">
                    <Icon name="notifications" className="text-xl" />
                </button>

                {/* User Avatar */}
                <div
                    className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10"
                    style={{ backgroundImage: "url('https://ui-avatars.com/api/?name=User+Admin&background=random')" }}
                ></div>
            </div>
        </header>
    );
};

export default TopNav;
