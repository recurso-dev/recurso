from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.post_auth_saml_tenant_id_acs_body import PostAuthSamlTenantIDAcsBody
from ...types import Response


def _get_kwargs(
    tenant_id: UUID,
    *,
    body: PostAuthSamlTenantIDAcsBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/auth/saml/{tenant_id}/acs".format(
            tenant_id=quote(str(tenant_id), safe=""),
        ),
    }

    _kwargs["data"] = body.to_dict()
    headers["Content-Type"] = "application/x-www-form-urlencoded"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | None:
    if response.status_code == 302:
        response_302 = cast(Any, None)
        return response_302

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = Error.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: PostAuthSamlTenantIDAcsBody,
) -> Response[Any | Error]:
    """Assertion Consumer Service (IdP posts the SAMLResponse here)

     Validates the SAMLResponse against the tenant's IdP certificate, extracts the email, and maps it to
    an EXISTING user in the tenant (no JIT provisioning). On success sets the `recurso_session` cookie
    and 302s to the dashboard. Unknown email → 403; disabled/unconfigured tenant → 404; invalid
    assertion → 401.

    Args:
        tenant_id (UUID):
        body (PostAuthSamlTenantIDAcsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        tenant_id=tenant_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: PostAuthSamlTenantIDAcsBody,
) -> Any | Error | None:
    """Assertion Consumer Service (IdP posts the SAMLResponse here)

     Validates the SAMLResponse against the tenant's IdP certificate, extracts the email, and maps it to
    an EXISTING user in the tenant (no JIT provisioning). On success sets the `recurso_session` cookie
    and 302s to the dashboard. Unknown email → 403; disabled/unconfigured tenant → 404; invalid
    assertion → 401.

    Args:
        tenant_id (UUID):
        body (PostAuthSamlTenantIDAcsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return sync_detailed(
        tenant_id=tenant_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    tenant_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: PostAuthSamlTenantIDAcsBody,
) -> Response[Any | Error]:
    """Assertion Consumer Service (IdP posts the SAMLResponse here)

     Validates the SAMLResponse against the tenant's IdP certificate, extracts the email, and maps it to
    an EXISTING user in the tenant (no JIT provisioning). On success sets the `recurso_session` cookie
    and 302s to the dashboard. Unknown email → 403; disabled/unconfigured tenant → 404; invalid
    assertion → 401.

    Args:
        tenant_id (UUID):
        body (PostAuthSamlTenantIDAcsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        tenant_id=tenant_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: PostAuthSamlTenantIDAcsBody,
) -> Any | Error | None:
    """Assertion Consumer Service (IdP posts the SAMLResponse here)

     Validates the SAMLResponse against the tenant's IdP certificate, extracts the email, and maps it to
    an EXISTING user in the tenant (no JIT provisioning). On success sets the `recurso_session` cookie
    and 302s to the dashboard. Unknown email → 403; disabled/unconfigured tenant → 404; invalid
    assertion → 401.

    Args:
        tenant_id (UUID):
        body (PostAuthSamlTenantIDAcsBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return (
        await asyncio_detailed(
            tenant_id=tenant_id,
            client=client,
            body=body,
        )
    ).parsed
