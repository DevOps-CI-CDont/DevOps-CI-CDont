import { authenticatedRouter, router } from "@/globals/router";
import { getLogout } from "@/server/getLogout";
import useAuthStore from "@/store/authStore";
import Link from "next/link";
import { useRouter } from "next/router";
import { useCookies } from "react-cookie";
import { DarkModeToggle } from "../DarkModeToggle";
import { useGetUsernameById } from "@/hooks/useGetUsernameById";

export function Header() {
	const isAuth = useAuthStore((state) => state.isAuth);
	const [userIdCookie, , removeUserIdCookie] = useCookies(["user_id"]);
	const nextRouter = useRouter();
	const { data: username } = useGetUsernameById(userIdCookie.user_id as string);

	return (
		<div className="border-b dark:border-b-slate-800 shadow-md w-screen fixed top-0 left-0 right-0 bg-white dark:bg-slate-900 dark:text-slate-100">
			<nav className="flex justify-between items-center h-20 max-w-7xl mx-auto px-2">
				<div>
					<Link href={"/public"}>
						<h2 className="font-bold text-lg">ITU Minitwit (now with CD)!</h2>
					</Link>
					{username && <span>Welcome: {username.toString()}</span>}
				</div>
				<ul className="flex justify-center items-center">
					<DarkModeToggle />
					{isAuth ? (
						<>
							{authenticatedRouter.map((route) => {
								return (
									<li key={route.id} className="mx-2 hover:underline">
										<Link href={route.path || ""} prefetch={false}>
											{route.text}
										</Link>
									</li>
								);
							})}
							<li
								className="mx-2 hover:underline cursor-pointer"
								onClick={handleSignOut}
							>
								Sign out
							</li>
						</>
					) : (
						router.map((route) => {
							return (
								<li key={route.id} className="mx-2 hover:underline">
									<Link href={route.path || ""} prefetch={false}>
										{route.text}
									</Link>
								</li>
							);
						})
					)}
				</ul>
			</nav>
		</div>
	);

	async function handleSignOut() {
		await getLogout();
		removeUserIdCookie("user_id");
		nextRouter.push("/public");
	}
}
