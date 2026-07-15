from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...types import Response


def _get_kwargs(
    id: UUID,
    addon_id: UUID,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/v1/subscriptions/{id}/addons/{addon_id}".format(
            id=quote(str(id), safe=""),
            addon_id=quote(str(addon_id), safe=""),
        ),
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error | None:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

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
    id: UUID,
    addon_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | Error]:
    """Detach an add-on from a subscription

     Removes the add-on from the subscription. The change takes effect from the next recurring invoice.

    Args:
        id (UUID):
        addon_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        addon_id=addon_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    addon_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Any | Error | None:
    """Detach an add-on from a subscription

     Removes the add-on from the subscription. The change takes effect from the next recurring invoice.

    Args:
        id (UUID):
        addon_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return sync_detailed(
        id=id,
        addon_id=addon_id,
        client=client,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    addon_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | Error]:
    """Detach an add-on from a subscription

     Removes the add-on from the subscription. The change takes effect from the next recurring invoice.

    Args:
        id (UUID):
        addon_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        addon_id=addon_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    addon_id: UUID,
    *,
    client: AuthenticatedClient | Client,
) -> Any | Error | None:
    """Detach an add-on from a subscription

     Removes the add-on from the subscription. The change takes effect from the next recurring invoice.

    Args:
        id (UUID):
        addon_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
    """

    return (
        await asyncio_detailed(
            id=id,
            addon_id=addon_id,
            client=client,
        )
    ).parsed
