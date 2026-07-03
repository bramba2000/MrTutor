import { Alert, Button, Paper, TextInput } from "@mantine/core";
import { useNavigate } from "@tanstack/react-router";
import { TitleLayout } from "../components/TitleLayout";
import { Route } from "#/routes/auth/register";
import classes from "./AuthPage.module.css";
import { useRegister } from "../mutations";
import { CustomLink } from "#/components/CustomLink";
import { useForm } from "@mantine/form";
import type { RegisterCredentials } from "../api";
import { isUserRole, safeRedirect, UserRole } from "../constants";
import { problemsToFieldErrors, validationProblems } from "#/lib/api";
import { useState } from "react";

interface searchParams {
  redirect: string;
  role: UserRole;
}

export function validateSearchParams(
  search: Record<string, unknown>,
): searchParams {
  return {
    redirect: safeRedirect(search.redirect),
    role: isUserRole(search.role) ? search.role : UserRole.Student,
  };
}

export function RegisterPage() {
  const { redirect, role } = Route.useSearch();
  const navigate = useNavigate();
  const register = useRegister();
  const [globalError, setGlobalError] = useState("");
  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      username: "",
      email: "",
      password: "",
      passwordConfirm: "",
    } as RegisterCredentials & {
      passwordConfirm: string;
    },
    validate: {
      username: (value) =>
        value.length < 3 ? "Username must be at least 3 characters" : null,
      email: (value) => (/^\S+@\S+$/.test(value) ? null : "Invalid email"),
      password: (value) =>
        value.length < 6 ? "Password must be at least 6 characters" : null,
      passwordConfirm: (value, values) =>
        value !== values.password ? "Passwords do not match" : null,
    },
  });

  function handleError(error: Error) {
    const problems = validationProblems(error);
    if (problems) {
      const fieldErrors = problemsToFieldErrors(problems);
      form.setErrors(fieldErrors);
      if (fieldErrors.role) {
        setGlobalError(fieldErrors.role);
      }
    }
  }

  function handleSubmit(values: typeof form.values) {
    register.mutate(
      {
        username: values.username,
        email: values.email,
        password: values.password,
        role: role,
      },
      {
        onSuccess: () => navigate({ href: redirect }),
        onError: handleError,
      },
    );
  }

  return (
    <TitleLayout
      title="Register"
      subtitle="You already have an account?"
      subtitleLink={
        <CustomLink to={"/auth/login"} search={{ redirect }}>
          Login with it
        </CustomLink>
      }
    >
      <form
        className={classes["auth-form"]}
        onSubmit={form.onSubmit(handleSubmit)}
      >
        <Alert
          variant="light"
          color="red"
          title="Error"
          hidden={!globalError}
          style={{ textTransform: "capitalize" }}
        >
          {globalError}
        </Alert>
        <TextInput
          label="Username"
          placeholder="Enter your username"
          withAsterisk
          autoComplete="username webauthn"
          {...form.getInputProps("username")}
        />
        <TextInput
          label="Email"
          placeholder="Enter your email"
          withAsterisk
          type="email"
          autoComplete="email webauthn"
          {...form.getInputProps("email")}
        />
        <TextInput
          label="Password"
          placeholder="Enter your password"
          withAsterisk
          type="password"
          autoComplete="new-password webauthn"
          {...form.getInputProps("password")}
        />
        <TextInput
          label="Confirm password"
          placeholder="Enter your password"
          withAsterisk
          type="password"
          autoComplete="new-password webauthn"
          {...form.getInputProps("passwordConfirm")}
        />
        <Button
          type="submit"
          loading={register.isPending}
          disabled={register.isPending}
        >
          Submit
        </Button>
      </form>
    </TitleLayout>
  );
}
