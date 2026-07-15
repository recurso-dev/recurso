from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.put_v1_sso_connection_response_200 import PutV1SsoConnectionResponse200
from ...models.sso_connection_upsert_request import SSOConnectionUpsertRequest
from ...types import Response


def _get_kwargs(
    *,
    body: SSOConnectionUpsertRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/v1/sso/connection",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | PutV1SsoConnectionResponse200 | None:
    if response.status_code == 200:
        response_200 = PutV1SsoConnectionResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = Error.from_dict(response.json())

        return response_403

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | PutV1SsoConnectionResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: SSOConnectionUpsertRequest,
) -> Response[Error | PutV1SsoConnectionResponse200]:
    """Create or update the tenant's SAML IdP configuration (owner/admin only)

     Upserts the tenant's IdP config. Provide either `idp_metadata_xml`, or all of `idp_entity_id` +
    `idp_sso_url` + `idp_certificate`. Enabling a connection that is not fully configured is rejected.
    Members get 403.

    Args:
        body (SSOConnectionUpsertRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutV1SsoConnectionResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    body: SSOConnectionUpsertRequest,
) -> Error | PutV1SsoConnectionResponse200 | None:
    """Create or update the tenant's SAML IdP configuration (owner/admin only)

     Upserts the tenant's IdP config. Provide either `idp_metadata_xml`, or all of `idp_entity_id` +
    `idp_sso_url` + `idp_certificate`. Enabling a connection that is not fully configured is rejected.
    Members get 403.

    Args:
        body (SSOConnectionUpsertRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutV1SsoConnectionResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: SSOConnectionUpsertRequest,
) -> Response[Error | PutV1SsoConnectionResponse200]:
    """Create or update the tenant's SAML IdP configuration (owner/admin only)

     Upserts the tenant's IdP config. Provide either `idp_metadata_xml`, or all of `idp_entity_id` +
    `idp_sso_url` + `idp_certificate`. Enabling a connection that is not fully configured is rejected.
    Members get 403.

    Args:
        body (SSOConnectionUpsertRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PutV1SsoConnectionResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    body: SSOConnectionUpsertRequest,
) -> Error | PutV1SsoConnectionResponse200 | None:
    """Create or update the tenant's SAML IdP configuration (owner/admin only)

     Upserts the tenant's IdP config. Provide either `idp_metadata_xml`, or all of `idp_entity_id` +
    `idp_sso_url` + `idp_certificate`. Enabling a connection that is not fully configured is rejected.
    Members get 403.

    Args:
        body (SSOConnectionUpsertRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PutV1SsoConnectionResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
