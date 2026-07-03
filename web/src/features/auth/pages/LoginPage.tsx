import { Button, TextInput } from "@mantine/core";
import { useNavigate } from "@tanstack/react-router";
import { TitleLayout } from "../components/TitleLayout";
import { Route } from "#/routes/auth/login";
import classes from "./AuthPage.module.css";
import { useLogin } from "../mutations";
import { CustomLink } from "#/components/CustomLink";
import { safeRedirect, UserRole } from "../constants";

interface LoginPageSearchParams {
  redirect: string;
}

export function validateSearchParams(
  search: Record<string, unknown>,
): LoginPageSearchParams {
  return {
    redirect: safeRedirect(search.redirect),
  };
}

export function LoginPage() {
  const { redirect } = Route.useSearch();
  const navigate = useNavigate();
  const login = useLogin();

  function handleSubmit(event: React.SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    login.mutate(
      {
        password: formData.get("password") as string,
        token: formData.get("email") as string,
      },
      {
        onSuccess: () => navigate({ href: redirect }),
      },
    );
  }

  return (
    <TitleLayout
      title="Login"
      subtitle="You don't have an account yet?"
      subtitleLink={
        <CustomLink
          to={"/auth/register"}
          search={{ redirect, role: UserRole.Student }}
        >
          Create one
        </CustomLink>
      }
    >
      <form className={classes["auth-form"]} onSubmit={handleSubmit}>
        <TextInput
          label="Email"
          placeholder="Enter your email"
          required
          type="email"
          name="email"
        />
        <TextInput
          label="Password"
          placeholder="Enter your password"
          required
          type="password"
          name="password"
        />
        <Button
          type="submit"
          loading={login.isPending}
          disabled={login.isPending}
        >
          Submit
        </Button>
      </form>
    </TitleLayout>
  );
}
