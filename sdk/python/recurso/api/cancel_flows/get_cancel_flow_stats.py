from http import HTTPStatus
from typing import Any
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.flow_stats import FlowStats
from ...types import UNSET, Response


def _get_kwargs(
    *,
    flow_id: UUID,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_flow_id = str(flow_id)
    params["flow_id"] = json_flow_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/cancel-flows/stats",
        "params": params,
    }

    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | FlowStats | None:
    if response.status_code == 200:
        response_200 = FlowStats.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | FlowStats]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    flow_id: UUID,
) -> Response[Error | FlowStats]:
    """Cancel flow analytics

    Args:
        flow_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FlowStats]
    """

    kwargs = _get_kwargs(
        flow_id=flow_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    flow_id: UUID,
) -> Error | FlowStats | None:
    """Cancel flow analytics

    Args:
        flow_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FlowStats
    """

    return sync_detailed(
        client=client,
        flow_id=flow_id,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    flow_id: UUID,
) -> Response[Error | FlowStats]:
    """Cancel flow analytics

    Args:
        flow_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FlowStats]
    """

    kwargs = _get_kwargs(
        flow_id=flow_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    flow_id: UUID,
) -> Error | FlowStats | None:
    """Cancel flow analytics

    Args:
        flow_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FlowStats
    """

    return (
        await asyncio_detailed(
            client=client,
            flow_id=flow_id,
        )
    ).parsed
