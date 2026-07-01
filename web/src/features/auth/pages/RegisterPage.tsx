import { Button, TextInput } from "@mantine/core";
import { TitleLayout } from "../components/TitleLayout";
import { Route } from "#/routes/auth/register";
import classes from "./AuthPage.module.css";
import { useRegister } from "../mutations";
import { CustomLink } from "#/components/CustomLink";
import { useForm } from "@mantine/form";

export function RegisterPage() {
  const { redirect } = Route.useSearch();
  const navigate = Route.useNavigate();
  const register = useRegister();
  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      username: "",
      email: "",
      password: "",
      passwordConfirm: "",
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

  function handleSubmit(values: typeof form.values) {
    register.mutate(
      {
        Username: values.username,
        Email: values.email,
        Password: values.password,
      },
      {
        onSuccess: () => navigate({ to: redirect ?? "/" }),
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
